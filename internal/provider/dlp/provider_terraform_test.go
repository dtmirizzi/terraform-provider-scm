package provider

// Terraform-level tests for the DLP resources.
//
// Unlike the CRUD unit tests in this package, these drive a real Terraform
// binary through plan -> apply -> refresh -> plan (and update / import steps)
// against the in-process mock DLP backends. That exercises Terraform core's
// plan/apply consistency checks ("Provider produced inconsistent result after
// apply", non-empty plans after apply, import drift), which the direct CRUD
// tests cannot catch.
//
// A Terraform CLI is required. It is located via TF_ACC_TERRAFORM_PATH or
// `terraform` on PATH; when neither is available the tests are skipped so
// `go test ./...` stays hermetic.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	pschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	rt "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/paloaltonetworks/terraform-provider-scm/internal/provider/dlp/client"
)

// dlpTestProvider is a minimal "scm" provider exposing only the DLP package,
// wired to a DLP client that points at the mock backends.
type dlpTestProvider struct{ client *dlpclient.Client }

func (p *dlpTestProvider) Metadata(_ context.Context, _ fwprovider.MetadataRequest, resp *fwprovider.MetadataResponse) {
	resp.TypeName = "scm"
}

func (p *dlpTestProvider) Schema(_ context.Context, _ fwprovider.SchemaRequest, resp *fwprovider.SchemaResponse) {
	resp.Schema = pschema.Schema{}
}

func (p *dlpTestProvider) Configure(_ context.Context, _ fwprovider.ConfigureRequest, resp *fwprovider.ConfigureResponse) {
	clients := map[string]any{"dlp": p.client}
	resp.ResourceData = clients
	resp.DataSourceData = clients
	resp.ActionData = clients
}

func (p *dlpTestProvider) Resources(context.Context) []func() resource.Resource {
	return GetResources()
}

func (p *dlpTestProvider) DataSources(context.Context) []func() datasource.DataSource {
	return GetDataSources()
}

func (p *dlpTestProvider) Actions(context.Context) []func() action.Action {
	return GetActions()
}

type dlpTestEnv struct {
	factories  map[string]func() (tfprotov6.ProviderServer, error)
	primitives *mockPrimitivesBackend
}

// newDlpTestEnv starts every mock backend behind a single front server that
// routes each request to the first backend that does not answer 404/405.
func newDlpTestEnv(t *testing.T) *dlpTestEnv {
	t.Helper()
	if os.Getenv("TF_ACC_TERRAFORM_PATH") == "" {
		if _, err := exec.LookPath("terraform"); err != nil {
			t.Skip("terraform CLI not found; set TF_ACC_TERRAFORM_PATH or add terraform to PATH to run Terraform-level DLP tests")
		}
	}

	primitives := newMockBackend()
	primitivesSrv := setupTestServer(primitives)
	profilesSrv, _, _ := setupProfilesMockServer(t)
	policiesSrv := setupPoliciesMockServer(newMockPoliciesBackend())
	tenantSrv, _, _ := setupMockServer(t)
	backends := []*httptest.Server{primitivesSrv, profilesSrv, policiesSrv, tenantSrv}

	front := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		for i, backend := range backends {
			req, err := http.NewRequest(r.Method, backend.URL+r.URL.RequestURI(), bytes.NewReader(body))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			req.Header = r.Header.Clone()
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				continue
			}
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			last := i == len(backends)-1
			if (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed) && !last {
				continue
			}
			for k, v := range resp.Header {
				w.Header()[k] = v
			}
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write(respBody)
			return
		}
		http.Error(w, "no mock backend reachable", http.StatusBadGateway)
	}))
	t.Cleanup(func() {
		front.Close()
		for _, b := range backends {
			b.Close()
		}
	})

	client := dlpclient.NewClient(dlpclient.Config{Host: front.URL, HttpClient: front.Client()})
	return &dlpTestEnv{
		factories: map[string]func() (tfprotov6.ProviderServer, error){
			"scm": providerserver.NewProtocol6WithError(&dlpTestProvider{client: client}),
		},
		primitives: primitives,
	}
}

func (e *dlpTestEnv) run(t *testing.T, steps ...rt.TestStep) {
	t.Helper()
	rt.UnitTest(t, rt.TestCase{ProtoV6ProviderFactories: e.factories, Steps: steps})
}

func importStep(addr string, ignore ...string) rt.TestStep {
	return rt.TestStep{ResourceName: addr, ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: ignore}
}

// ---------------------------------------------------------------------------
// Primitives
// ---------------------------------------------------------------------------

func TestTerraformDataPattern(t *testing.T) {
	env := newDlpTestEnv(t)
	env.run(t,
		rt.TestStep{Config: `
resource "scm_dlp_data_pattern" "p" {
  name        = "Custom-Employee-ID"
  description = "Detects EMP-XXXXXX"
  type        = "custom"
  detection_config = {
    technique                   = "weighted_regex"
    supported_confidence_levels = ["low", "medium", "high"]
  }
  matching_rules = {
    proximity_distance = 50
    regexes            = [{ regex = "EMP-[0-9]{6}", weight = 10 }]
  }
}`,
			Check: rt.ComposeAggregateTestCheckFunc(
				rt.TestCheckResourceAttrSet("scm_dlp_data_pattern.p", "id"),
				rt.TestCheckResourceAttr("scm_dlp_data_pattern.p", "matching_rules.regexes.0.weight", "10"),
			),
		},
		rt.TestStep{Config: `
resource "scm_dlp_data_pattern" "p" {
  name = "Custom-Employee-ID"
  type = "custom"
  detection_config = {
    technique                   = "weighted_regex"
    supported_confidence_levels = ["high"]
  }
  matching_rules = {
    regexes = [{ regex = "EMP-[0-9]{6}", weight = 5 }]
  }
}`,
			Check: rt.TestCheckResourceAttr("scm_dlp_data_pattern.p", "matching_rules.regexes.0.weight", "5"),
		},
		importStep("scm_dlp_data_pattern.p"),
	)
}

const dictionaryWithKeywords = `
resource "scm_dlp_dictionary" "d" {
  name               = "Internal-Project-Codenames"
  description        = "first"
  category           = "Confidential"
  region_name        = "us"
  original_file_name = "codenames.txt"
  keyword_list       = ["ProjectApollo", "ProjectOrion"]
}`

func TestTerraformDictionary(t *testing.T) {
	env := newDlpTestEnv(t)
	env.run(t,
		rt.TestStep{
			Config: dictionaryWithKeywords,
			Check:  rt.TestCheckResourceAttr("scm_dlp_dictionary.d", "keyword_list.#", "2"),
		},
		// Changing only metadata while keyword_list is omitted must keep the keywords.
		rt.TestStep{Config: `
resource "scm_dlp_dictionary" "d" {
  name               = "Internal-Project-Codenames"
  description        = "second"
  category           = "Confidential"
  region_name        = "us"
  original_file_name = "codenames.txt"
}`,
			Check: rt.ComposeAggregateTestCheckFunc(
				rt.TestCheckResourceAttr("scm_dlp_dictionary.d", "description", "second"),
				rt.TestCheckResourceAttr("scm_dlp_dictionary.d", "keyword_list.#", "2"),
			),
		},
		rt.TestStep{Config: `
resource "scm_dlp_dictionary" "d" {
  name               = "Internal-Project-Codenames"
  description        = "second"
  category           = "Confidential"
  region_name        = "us"
  original_file_name = "codenames.txt"
  keyword_list       = ["ProjectApollo", "ProjectOrion", "ProjectZeus"]
}`,
			Check: rt.TestCheckResourceAttr("scm_dlp_dictionary.d", "keyword_list.#", "3"),
		},
		importStep("scm_dlp_dictionary.d"),
	)
}

func TestTerraformDictionaryRejectsPaddedKeywords(t *testing.T) {
	env := newDlpTestEnv(t)
	env.run(t, rt.TestStep{
		Config: `
resource "scm_dlp_dictionary" "d" {
  name               = "padded"
  category           = "Confidential"
  region_name        = "us"
  original_file_name = "codenames.txt"
  keyword_list       = [" ProjectApollo"]
}`,
		ExpectError: regexp.MustCompile(`leading or trailing\s+whitespace`),
	})
}

func TestTerraformDocumentType(t *testing.T) {
	env := newDlpTestEnv(t)
	env.run(t,
		rt.TestStep{Config: `
resource "scm_dlp_document_type" "d" {
  name                = "contracts"
  category            = "Legal"
  detection_technique = "document_fingerprint"
  original_file_name  = "contract.pdf"
}`},
		importStep("scm_dlp_document_type.d"),
	)
}

func TestTerraformEdmDatasetAdoptAndDestroyKeepsData(t *testing.T) {
	env := newDlpTestEnv(t)
	id := "edm-tf-1"
	status := "COMPLETED"
	env.primitives.edmDatasets[id] = dlpclient.EdmDatasetDTO{ID: &id, Name: "Customer_PII", IndexStatus: &status}

	rt.UnitTest(t, rt.TestCase{
		ProtoV6ProviderFactories: env.factories,
		// Destroy only removes the adopted dataset from state; the uploaded data must survive.
		CheckDestroy: func(*terraform.State) error {
			env.primitives.mu.Lock()
			defer env.primitives.mu.Unlock()
			if _, ok := env.primitives.edmDatasets[id]; !ok {
				return fmt.Errorf("EDM dataset %s was deleted from the backend on destroy", id)
			}
			return nil
		},
		Steps: []rt.TestStep{
			{
				Config: `resource "scm_dlp_edm_dataset" "e" { name = "Customer_PII" }`,
				Check:  rt.TestCheckResourceAttr("scm_dlp_edm_dataset.e", "id", id),
			},
			importStep("scm_dlp_edm_dataset.e"),
		},
	})
}

// ---------------------------------------------------------------------------
// Profiles
// ---------------------------------------------------------------------------

func dataProfileConfig(count int) string {
	return `
resource "scm_dlp_data_profile" "p" {
  name = "PCI"
  detection_rules = [{
    rule_type = "expression_tree"
    expression_tree = {
      operator_type = "or"
      sub_expressions = [
        { rule_item = { id = "abc", detection_technique = "regex", occurrence_operator_type = "more_than_equal_to", occurrence_count = ` + strconv.Itoa(count) + ` } },
        { operator_type = "and", sub_expressions = [
          { rule_item = { id = "def", detection_technique = "weighted_regex", occurrence_operator_type = "any" } },
          { rule_item = { id = "ghi", detection_technique = "dictionary", occurrence_operator_type = "any" } }
        ] }
      ]
    }
  }]
}`
}

func TestTerraformDataProfile(t *testing.T) {
	env := newDlpTestEnv(t)
	env.run(t,
		rt.TestStep{
			Config: dataProfileConfig(5),
			Check:  rt.TestCheckResourceAttr("scm_dlp_data_profile.p", "version", "1"),
		},
		// The backend bumps the version on every update; the plan must not pin it.
		rt.TestStep{
			Config: dataProfileConfig(7),
			Check: rt.ComposeAggregateTestCheckFunc(
				rt.TestCheckResourceAttr("scm_dlp_data_profile.p", "version", "2"),
				rt.TestCheckResourceAttr("scm_dlp_data_profile.p", "detection_rules.0.expression_tree.sub_expressions.0.rule_item.occurrence_count", "7"),
			),
		},
		importStep("scm_dlp_data_profile.p", "detection_rules.0.expression_tree.expression_tree_json"),
	)
}

func TestTerraformDataProfileExpressionTreeJSON(t *testing.T) {
	env := newDlpTestEnv(t)
	env.run(t, rt.TestStep{Config: `
resource "scm_dlp_data_profile" "p" {
  name = "PCI-json"
  detection_rules = [{
    rule_type = "expression_tree"
    expression_tree = {
      expression_tree_json = jsonencode({
        operator_type   = "or"
        sub_expressions = [{ rule_item = { id = "abc", detection_technique = "regex", occurrence_operator_type = "any" } }]
      })
    }
  }]
}`})
}

func TestTerraformDataFilteringProfile(t *testing.T) {
	env := newDlpTestEnv(t)
	cfg := func(action string) string {
		return dataProfileConfig(5) + `
resource "scm_dlp_data_filtering_profile" "f" {
  data_profile_id = tonumber(scm_dlp_data_profile.p.id)
  direction       = "both"
  scan_type       = "include"
  file_based      = true
  non_file_based  = true
  file_type       = ["pdf", "csv"]
  log_severity    = "HIGH"
  rule1_action    = "` + action + `"
}`
	}
	env.run(t,
		rt.TestStep{Config: cfg("block")},
		rt.TestStep{
			Config: cfg("alert"),
			Check:  rt.TestCheckResourceAttr("scm_dlp_data_filtering_profile.f", "rule1_action", "alert"),
		},
		importStep("scm_dlp_data_filtering_profile.f"),
	)
}

// ---------------------------------------------------------------------------
// Tenant configuration
// ---------------------------------------------------------------------------

func TestTerraformTenantSettings(t *testing.T) {
	env := newDlpTestEnv(t)
	cfg := func(mask string, minutes int) string {
		return `
resource "scm_dlp_tenant_config" "g" {
  enable_snippets = true
  mask_level      = "` + mask + `"
}

resource "scm_dlp_alert_settings" "a" {
  incident_notification_enablement   = true
  detection_exemption_period_minutes = ` + strconv.Itoa(minutes) + `
}

resource "scm_dlp_ocr_enablement" "o" {
  target_service_name = "prisma-access"
  ocr_enablement      = "ENABLED"
}`
	}
	env.run(t,
		rt.TestStep{Config: cfg("PARTIAL_MASK", 60)},
		rt.TestStep{
			Config: cfg("FULL_MASK", 120),
			Check: rt.ComposeAggregateTestCheckFunc(
				rt.TestCheckResourceAttr("scm_dlp_tenant_config.g", "mask_level", "FULL_MASK"),
				rt.TestCheckResourceAttr("scm_dlp_alert_settings.a", "detection_exemption_period_minutes", "120"),
			),
		},
		importStep("scm_dlp_tenant_config.g"),
		rt.TestStep{
			ResourceName:                         "scm_dlp_ocr_enablement.o",
			ImportState:                          true,
			ImportStateId:                        "prisma-access",
			ImportStateVerify:                    true,
			ImportStateVerifyIdentifierAttribute: "target_service_name",
		},
	)
}

// ---------------------------------------------------------------------------
// Policies
// ---------------------------------------------------------------------------

const endpointPolicyUpdated = `
resource "scm_dlp_endpoint_policy" "e" {
  name          = "ep"
  type          = "endpoint_peripheral_control"
  severity      = "medium"
  enabled       = false
  scope_json    = "{\"users\": [\"a\"]}"
  response_json = jsonencode({ action = "alert" })
}`

func TestTerraformEndpointPolicy(t *testing.T) {
	env := newDlpTestEnv(t)
	env.run(t,
		// Non-compact JSON must not produce an inconsistent result or a diff.
		rt.TestStep{Config: `
resource "scm_dlp_endpoint_policy" "e" {
  name          = "ep"
  type          = "endpoint_peripheral_control"
  severity      = "high"
  enabled       = true
  scope_json    = "{\"users\": [\"a\"]}"
  response_json = jsonencode({ action = "block" })
}`},
		rt.TestStep{
			Config: endpointPolicyUpdated,
			Check:  rt.TestCheckResourceAttr("scm_dlp_endpoint_policy.e", "severity", "medium"),
		},
		// Imported JSON is the backend's compact form: semantically equal to the
		// configured value but not byte-identical, so skip the strict comparison.
		importStep("scm_dlp_endpoint_policy.e", "scope_json"),
	)
}

func TestTerraformPeripheral(t *testing.T) {
	env := newDlpTestEnv(t)
	env.run(t,
		rt.TestStep{Config: `
resource "scm_dlp_peripheral" "p" {
  name       = "usb1"
  type       = "usb"
  vendor_id  = "0781"
  product_id = "5581"
}`},
		rt.TestStep{Config: `
resource "scm_dlp_peripheral" "p" {
  name       = "usb1"
  type       = "usb"
  product_id = "5581"
}`},
		importStep("scm_dlp_peripheral.p"),
	)
}

func TestTerraformSyslogProfile(t *testing.T) {
	env := newDlpTestEnv(t)
	cfg := func(port int) string {
		return `
resource "scm_dlp_syslog_profile" "s" {
  profile_name = "sys"
  address      = "10.0.0.1"
  port         = ` + strconv.Itoa(port) + `
  alert_email  = "alerts@example.com"
  format       = "CEF"
}`
	}
	env.run(t,
		rt.TestStep{Config: cfg(514)},
		rt.TestStep{Config: cfg(6514), Check: rt.TestCheckResourceAttr("scm_dlp_syslog_profile.s", "port", "6514")},
		importStep("scm_dlp_syslog_profile.s"),
	)
}

func TestTerraformForwardProfile(t *testing.T) {
	env := newDlpTestEnv(t)
	env.run(t,
		rt.TestStep{Config: `
resource "scm_dlp_forward_profile" "f" {
  name    = "fwd"
  enabled = true
  regions = ["us", "eu"]
}`},
		rt.TestStep{Config: `
resource "scm_dlp_forward_profile" "f" {
  name    = "fwd"
  enabled = false
}`},
		importStep("scm_dlp_forward_profile.f"),
	)
}

func TestTerraformDryRunScanPositiveAndNegativeSuite(t *testing.T) {
	env := newDlpTestEnv(t)
	env.run(t,
		rt.TestStep{
			Config: `
resource "scm_dlp_data_profile" "prof" {
  name = "emp-id-profile"
  detection_rules = [{
    rule_type = "expression_tree"
    expression_tree = {
      operator_type = "or"
      sub_expressions = [{
        rule_item = {
          id                       = "pat-123"
          detection_technique      = "weighted_regex"
          occurrence_operator_type = "any"
        }
      }]
    }
  }]
}

data "scm_dlp_dry_run_scan" "suite" {
  data_profile_id  = scm_dlp_data_profile.prof.id
  fail_on_mismatch = true

  test_cases = [
    {
      name             = "positive-employee-id-leak"
      file_content     = "Confidential record for employee EMP-123456."
      file_name        = "positive_leak.txt"
      expected_verdict = "MATCHED"
      expected_action  = "block"
    },
    {
      name             = "negative-benign-document"
      file_content     = "Public marketing announcement with no sensitive identifiers."
      file_name        = "negative_public.txt"
      expected_verdict = "NOT_MATCHED"
      expected_action  = "allow"
    }
  ]
}`,
			Check: rt.ComposeAggregateTestCheckFunc(
				rt.TestCheckResourceAttr("data.scm_dlp_dry_run_scan.suite", "passed", "true"),
				rt.TestCheckResourceAttr("data.scm_dlp_dry_run_scan.suite", "results.#", "2"),
				rt.TestCheckResourceAttr("data.scm_dlp_dry_run_scan.suite", "results.0.name", "positive-employee-id-leak"),
				rt.TestCheckResourceAttr("data.scm_dlp_dry_run_scan.suite", "results.0.verdict", "MATCHED"),
				rt.TestCheckResourceAttr("data.scm_dlp_dry_run_scan.suite", "results.0.passed", "true"),
				rt.TestCheckResourceAttr("data.scm_dlp_dry_run_scan.suite", "results.1.name", "negative-benign-document"),
				rt.TestCheckResourceAttr("data.scm_dlp_dry_run_scan.suite", "results.1.verdict", "NOT_MATCHED"),
				rt.TestCheckResourceAttr("data.scm_dlp_dry_run_scan.suite", "results.1.passed", "true"),
			),
		},
	)
}
