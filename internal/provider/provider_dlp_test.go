package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDlpProviderRegistration(t *testing.T) {
	ctx := context.Background()
	p := New("1.0.0")

	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected provider schema diagnostics: %v", schemaResp.Diagnostics)
	}
	if _, ok := schemaResp.Schema.Attributes["dlp_host"]; !ok {
		t.Errorf("expected provider schema to contain dlp_host attribute")
	}
	if _, ok := schemaResp.Schema.Attributes["dlp_service_name"]; !ok {
		t.Errorf("expected provider schema to contain dlp_service_name attribute")
	}

	expectedResources := []string{
		"scm_dlp_tenant_config",
		"scm_dlp_ocr_enablement",
		"scm_dlp_alert_settings",
		"scm_dlp_data_pattern",
		"scm_dlp_dictionary",
		"scm_dlp_document_type",
		"scm_dlp_edm_dataset",
		"scm_dlp_data_profile",
		"scm_dlp_data_filtering_profile",
		"scm_dlp_endpoint_policy",
		"scm_dlp_peripheral",
		"scm_dlp_syslog_profile",
		"scm_dlp_forward_profile",
	}

	registeredResources := make(map[string]bool)
	for _, rf := range p.Resources(ctx) {
		r := rf()
		var metaResp resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "scm"}, &metaResp)
		if strings.HasPrefix(metaResp.TypeName, "scm_dlp_") {
			if registeredResources[metaResp.TypeName] {
				t.Errorf("duplicate DLP resource type name: %s", metaResp.TypeName)
			}
			registeredResources[metaResp.TypeName] = true

			var rSchemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &rSchemaResp)
			if rSchemaResp.Diagnostics.HasError() {
				t.Errorf("resource %s schema error: %v", metaResp.TypeName, rSchemaResp.Diagnostics)
			}
		}
	}

	for _, exp := range expectedResources {
		if !registeredResources[exp] {
			t.Errorf("expected DLP resource %q to be registered on ScmProvider", exp)
		}
	}

	expectedDataSources := []string{
		"scm_dlp_tenant_config",
		"scm_dlp_ocr_enablement",
		"scm_dlp_alert_settings",
		"scm_dlp_tenant_channels",
		"scm_dlp_data_pattern",
		"scm_dlp_data_pattern_list",
		"scm_dlp_dictionary",
		"scm_dlp_dictionary_list",
		"scm_dlp_document_type",
		"scm_dlp_document_type_list",
		"scm_dlp_edm_dataset",
		"scm_dlp_edm_dataset_list",
		"scm_dlp_data_profile",
		"scm_dlp_data_profile_list",
		"scm_dlp_data_filtering_profile",
		"scm_dlp_data_filtering_profile_list",
		"scm_dlp_endpoint_policy",
		"scm_dlp_endpoint_policy_list",
		"scm_dlp_peripheral",
		"scm_dlp_peripheral_list",
		"scm_dlp_syslog_profile",
		"scm_dlp_syslog_profile_list",
		"scm_dlp_forward_profile",
		"scm_dlp_forward_profile_list",
		"scm_dlp_dry_run_scan",
	}

	registeredDataSources := make(map[string]bool)
	for _, dsf := range p.DataSources(ctx) {
		ds := dsf()
		var metaResp datasource.MetadataResponse
		ds.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "scm"}, &metaResp)
		if strings.HasPrefix(metaResp.TypeName, "scm_dlp_") {
			if registeredDataSources[metaResp.TypeName] {
				t.Errorf("duplicate DLP data source type name: %s", metaResp.TypeName)
			}
			registeredDataSources[metaResp.TypeName] = true

			var dsSchemaResp datasource.SchemaResponse
			ds.Schema(ctx, datasource.SchemaRequest{}, &dsSchemaResp)
			if dsSchemaResp.Diagnostics.HasError() {
				t.Errorf("data source %s schema error: %v", metaResp.TypeName, dsSchemaResp.Diagnostics)
			}
		}
	}

	for _, exp := range expectedDataSources {
		if !registeredDataSources[exp] {
			t.Errorf("expected DLP data source %q to be registered on ScmProvider", exp)
		}
	}

	pa, ok := p.(provider.ProviderWithActions)
	if !ok {
		t.Fatalf("expected ScmProvider to implement ProviderWithActions")
	}

	expectedActions := []string{
		"scm_dlp_dry_run_scan",
	}
	registeredActions := make(map[string]bool)
	for _, af := range pa.Actions(ctx) {
		a := af()
		var metaResp action.MetadataResponse
		a.Metadata(ctx, action.MetadataRequest{ProviderTypeName: "scm"}, &metaResp)
		if strings.HasPrefix(metaResp.TypeName, "scm_dlp_") {
			registeredActions[metaResp.TypeName] = true
			var aSchemaResp action.SchemaResponse
			a.Schema(ctx, action.SchemaRequest{}, &aSchemaResp)
			if aSchemaResp.Diagnostics.HasError() {
				t.Errorf("action %s schema error: %v", metaResp.TypeName, aSchemaResp.Diagnostics)
			}
		}
	}
	for _, exp := range expectedActions {
		if !registeredActions[exp] {
			t.Errorf("expected DLP action %q to be registered on ScmProvider", exp)
		}
	}
}

func TestApplyDlpEnvFallback(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	dlpEnv := map[string]string{
		"DLP_AUTH_URL":      "https://dlp-auth.example",
		"DLP_CLIENT_ID":     "dlp-id",
		"DLP_CLIENT_SECRET": "dlp-secret",
		"DLP_TSG_ID":        "123",
	}

	t.Run("fills empty values from DLP_* env", func(t *testing.T) {
		a, id, sec, sc := applyDlpEnvFallback("", "", "", "", "", env(dlpEnv))
		if a != "https://dlp-auth.example" || id != "dlp-id" || sec != "dlp-secret" || sc != "tsg_id:123" {
			t.Fatalf("unexpected values: %q %q %q %q", a, id, sec, sc)
		}
	})

	t.Run("explicit config wins", func(t *testing.T) {
		a, id, sec, sc := applyDlpEnvFallback("https://cfg", "cfg-id", "cfg-secret", "tsg_id:9", "", env(dlpEnv))
		if a != "https://cfg" || id != "cfg-id" || sec != "cfg-secret" || sc != "tsg_id:9" {
			t.Fatalf("config was overridden: %q %q %q %q", a, id, sec, sc)
		}
	})

	t.Run("SCM_* env wins", func(t *testing.T) {
		m := map[string]string{"SCM_AUTH_URL": "x", "SCM_CLIENT_ID": "x", "SCM_CLIENT_SECRET": "x", "SCM_SCOPE": "x"}
		for k, v := range dlpEnv {
			m[k] = v
		}
		a, id, sec, sc := applyDlpEnvFallback("", "", "", "", "", env(m))
		if a != "" || id != "" || sec != "" || sc != "" {
			t.Fatalf("DLP_* env overrode SCM_* env: %q %q %q %q", a, id, sec, sc)
		}
	})

	t.Run("auth_file wins", func(t *testing.T) {
		a, id, sec, sc := applyDlpEnvFallback("", "", "", "", "/path/to/auth.json", env(dlpEnv))
		if a != "" || id != "" || sec != "" || sc != "" {
			t.Fatalf("DLP_* env overrode auth_file: %q %q %q %q", a, id, sec, sc)
		}
	})

	t.Run("tsg_id prefix is not doubled", func(t *testing.T) {
		_, _, _, sc := applyDlpEnvFallback("", "", "", "", "", env(map[string]string{"DLP_TSG_ID": "tsg_id:42"}))
		if sc != "tsg_id:42" {
			t.Fatalf("got %q", sc)
		}
	})
}
