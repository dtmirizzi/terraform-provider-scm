package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/paloaltonetworks/terraform-provider-scm/internal/provider/dlp/client"
)

// Ensure action and data source implementation interfaces are satisfied.
var (
	_ action.Action                      = &DryRunScanAction{}
	_ action.ActionWithConfigure         = &DryRunScanAction{}
	_ datasource.DataSource              = &DryRunScanDataSource{}
	_ datasource.DataSourceWithConfigure = &DryRunScanDataSource{}
)

const MaxUploadFileSize = 20 * 1024 * 1024 // 20 MB

func isPathRestricted(resolvedPath string) bool {
	sensitiveDirs := []string{"/etc", "/var", "/dev", "/usr", "/bin", "/sbin", "/opt", "/sys", "/proc", "/root"}
	tmp := filepath.Clean(os.TempDir())
	realTmp := tmp
	if eval, err := filepath.EvalSymlinks(tmp); err == nil {
		realTmp = filepath.Clean(eval)
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		sensitiveHome := []string{
			filepath.Join(home, ".ssh"),
			filepath.Join(home, ".aws"),
			filepath.Join(home, ".config"),
			filepath.Join(home, ".gnupg"),
			filepath.Join(home, ".netrc"),
		}
		for _, s := range sensitiveHome {
			sClean := filepath.Clean(s)
			if resolvedPath == sClean || strings.HasPrefix(resolvedPath, sClean+string(filepath.Separator)) {
				return true
			}
		}
	}

	for _, s := range sensitiveDirs {
		if resolvedPath == s || strings.HasPrefix(resolvedPath, s+string(filepath.Separator)) {
			if resolvedPath == tmp || strings.HasPrefix(resolvedPath, tmp+string(filepath.Separator)) ||
				resolvedPath == realTmp || strings.HasPrefix(resolvedPath, realTmp+string(filepath.Separator)) {
				return false
			}
			return true
		}
	}
	return false
}

// readValidatedUploadFile validates a local file path (regular file, non-empty, <=20MB, non-restricted path)
// and returns (fileBytes, basename, error).
func readValidatedUploadFile(rawPath string) ([]byte, string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return nil, "", fmt.Errorf("file_path must not be empty")
	}
	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return nil, "", fmt.Errorf("cannot resolve file_path %q: %w", rawPath, err)
	}
	resolved := filepath.Clean(absPath)
	if isPathRestricted(resolved) {
		return nil, "", fmt.Errorf("file_path %q resolves to a restricted system or credential directory", rawPath)
	}
	if eval, err := filepath.EvalSymlinks(absPath); err == nil {
		resolved = filepath.Clean(eval)
		if isPathRestricted(resolved) {
			return nil, "", fmt.Errorf("file_path %q resolves to a restricted system or credential directory", rawPath)
		}
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, "", fmt.Errorf("cannot access file %q: %w", rawPath, err)
	}
	if !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("file_path %q is not a regular file", rawPath)
	}
	if info.Size() == 0 {
		return nil, "", fmt.Errorf("file_path %q must not be empty", rawPath)
	}
	if info.Size() > MaxUploadFileSize {
		return nil, "", fmt.Errorf("file_path %q exceeds 20MB limit (413 Payload Too Large)", rawPath)
	}
	b, err := os.ReadFile(resolved)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read file %q: %w", rawPath, err)
	}
	return b, filepath.Base(resolved), nil
}

// normalizeDryRunDataProfileJSON ensures the dataProfile JSON has the internal DataProfileDTO
// fields required by POST /v2/api/dry-run/scan (dss-management POST /v1/{tenant}/data-profiles).
func normalizeDryRunDataProfileJSON(raw string) (string, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return "", err
	}
	if _, ok := m["type"]; !ok {
		m["type"] = "advanced"
	}
	pt, _ := m["profile_type"].(string)
	if pt == "" || pt == "advanced" {
		m["profile_type"] = "custom"
	}
	if _, ok := m["profile_status"]; !ok {
		m["profile_status"] = "active"
	}
	if _, ok := m["schema_version"]; !ok {
		m["schema_version"] = 2
	}
	if _, ok := m["created_by"]; !ok {
		m["created_by"] = "Strata Cloud Manager"
	}
	if _, ok := m["updated_by"]; !ok {
		m["updated_by"] = "Strata Cloud Manager"
	}
	delete(m, "is_granular_data_profile")
	out, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// resolveDryRunDataProfileJSON resolves the data profile JSON either from `data_profile` (raw JSON)
// or by fetching `data_profile_id` via GET /v2/api/data-profiles/{id}.
func resolveDryRunDataProfileJSON(ctx context.Context, client *dlpclient.Client, dataProfile, dataProfileId types.String) (string, string, error) {
	if !dataProfile.IsNull() && !dataProfile.IsUnknown() && strings.TrimSpace(dataProfile.ValueString()) != "" {
		norm, err := normalizeDryRunDataProfileJSON(dataProfile.ValueString())
		if err != nil {
			return "", "", fmt.Errorf("the 'data_profile' argument must be valid JSON: %w", err)
		}
		sourceID := ""
		if !dataProfileId.IsNull() && !dataProfileId.IsUnknown() {
			sourceID = strings.TrimSpace(dataProfileId.ValueString())
		}
		return norm, sourceID, nil
	}
	if !dataProfileId.IsNull() && !dataProfileId.IsUnknown() && strings.TrimSpace(dataProfileId.ValueString()) != "" {
		id := strings.TrimSpace(dataProfileId.ValueString())
		var rawDTO map[string]any
		if err := client.Get(ctx, fmt.Sprintf("/v2/api/data-profiles/%s", dlpclient.EscapeID(id)), nil, &rawDTO); err != nil {
			return "", "", fmt.Errorf("failed to fetch data profile %q: %w", id, err)
		}
		rawBytes, err := json.Marshal(rawDTO)
		if err != nil {
			return "", "", fmt.Errorf("failed to serialize data profile %q: %w", id, err)
		}
		norm, err := normalizeDryRunDataProfileJSON(string(rawBytes))
		if err != nil {
			return "", "", err
		}
		return norm, id, nil
	}
	return "", "", fmt.Errorf("either 'data_profile' (JSON string) or 'data_profile_id' (existing profile ID) must be provided")
}

// DryRunTestCaseModel represents a single positive or negative dry-run test case.
type DryRunTestCaseModel struct {
	Name            types.String `tfsdk:"name"`
	FilePath        types.String `tfsdk:"file_path"`
	FileContent     types.String `tfsdk:"file_content"`
	FileName        types.String `tfsdk:"file_name"`
	FileSha         types.String `tfsdk:"file_sha"`
	ExpectedVerdict types.String `tfsdk:"expected_verdict"`
	ExpectedAction  types.String `tfsdk:"expected_action"`
}

// DryRunTestCaseResultModel represents the computed result of a single dry-run test case.
type DryRunTestCaseResultModel struct {
	Name            types.String `tfsdk:"name"`
	FileName        types.String `tfsdk:"file_name"`
	FileSha         types.String `tfsdk:"file_sha"`
	Status          types.String `tfsdk:"status"`
	Verdict         types.String `tfsdk:"verdict"`
	Action          types.String `tfsdk:"action"`
	ExpectedVerdict types.String `tfsdk:"expected_verdict"`
	ExpectedAction  types.String `tfsdk:"expected_action"`
	Passed          types.Bool   `tfsdk:"passed"`
	FailureReason   types.String `tfsdk:"failure_reason"`
	ResultJson      types.String `tfsdk:"result_json"`
}

func extractDryRunFields(scanResult map[string]any) (status, verdict, actionStr string) {
	if st, ok := scanResult["status"].(string); ok {
		status = st
	}
	if vd, ok := scanResult["verdict"].(string); ok {
		verdict = vd
	}
	if act, ok := scanResult["action"].(string); ok {
		actionStr = act
	}
	if rpt, ok := scanResult["report"].(map[string]any); ok {
		if verdict == "" {
			if vd, ok := rpt["data_pattern_rule_1_verdict"].(string); ok && vd != "" {
				verdict = vd
			} else if vd2, ok := rpt["data_pattern_rule_2_verdict"].(string); ok && vd2 != "" {
				verdict = vd2
			}
		}
		if actionStr == "" {
			if act, ok := rpt["action"].(string); ok {
				actionStr = act
			}
		}
	}
	return status, verdict, actionStr
}

func executeSingleDryRunCase(
	ctx context.Context,
	client *dlpclient.Client,
	tc DryRunTestCaseModel,
	dataProfileStr string,
	sourceDataProfileID string,
	dryRunServiceName string,
) (DryRunTestCaseResultModel, error) {
	var out DryRunTestCaseResultModel
	caseName := "default"
	if !tc.Name.IsNull() && !tc.Name.IsUnknown() && strings.TrimSpace(tc.Name.ValueString()) != "" {
		caseName = strings.TrimSpace(tc.Name.ValueString())
	}
	out.Name = types.StringValue(caseName)

	var fileBytes []byte
	fileName := "sample.txt"

	if !tc.FilePath.IsNull() && !tc.FilePath.IsUnknown() && tc.FilePath.ValueString() != "" {
		b, baseName, err := readValidatedUploadFile(tc.FilePath.ValueString())
		if err != nil {
			return out, fmt.Errorf("test case %q: %w", caseName, err)
		}
		fileBytes = b
		fileName = baseName
	} else if !tc.FileContent.IsNull() && !tc.FileContent.IsUnknown() {
		fileBytes = []byte(tc.FileContent.ValueString())
		if len(fileBytes) == 0 || len(fileBytes) > MaxUploadFileSize {
			return out, fmt.Errorf("test case %q: file_content must be non-empty and at most 20MB", caseName)
		}
	} else {
		return out, fmt.Errorf("test case %q: either 'file_path' or 'file_content' must be provided", caseName)
	}

	if !tc.FileName.IsNull() && !tc.FileName.IsUnknown() && tc.FileName.ValueString() != "" {
		fileName = filepath.Base(tc.FileName.ValueString())
	}
	out.FileName = types.StringValue(fileName)

	fileSha := ""
	if !tc.FileSha.IsNull() && !tc.FileSha.IsUnknown() && tc.FileSha.ValueString() != "" {
		fileSha = tc.FileSha.ValueString()
	} else {
		hash := sha256.Sum256(fileBytes)
		fileSha = hex.EncodeToString(hash[:])
	}
	out.FileSha = types.StringValue(fileSha)

	parts := []dlpclient.MultipartPart{
		{
			FieldName: "fileSha",
			Data:      []byte(fileSha),
		},
		{
			FieldName: "dataProfile",
			Data:      []byte(dataProfileStr),
		},
		{
			FieldName:   "file",
			FileName:    fileName,
			ContentType: "application/octet-stream",
			Data:        fileBytes,
		},
	}
	if sourceDataProfileID != "" {
		parts = append(parts, dlpclient.MultipartPart{
			FieldName: "sourceDataProfileId",
			Data:      []byte(sourceDataProfileID),
		})
	}
	if dryRunServiceName != "" {
		parts = append(parts, dlpclient.MultipartPart{
			FieldName: "dryRunServiceName",
			Data:      []byte(dryRunServiceName),
		})
	}

	var scanResult map[string]any
	if err := client.DoMultipart(ctx, "POST", "/v2/api/dry-run/scan", nil, parts, &scanResult); err != nil {
		return out, fmt.Errorf("test case %q dry-run scan failed: %w", caseName, err)
	}

	rawJSON, _ := json.Marshal(scanResult)
	out.ResultJson = types.StringValue(string(rawJSON))

	status, verdict, actionStr := extractDryRunFields(scanResult)
	if status != "" {
		out.Status = types.StringValue(status)
	} else {
		out.Status = types.StringNull()
	}
	if verdict != "" {
		out.Verdict = types.StringValue(verdict)
	} else {
		out.Verdict = types.StringNull()
	}
	if actionStr != "" {
		out.Action = types.StringValue(actionStr)
	} else {
		out.Action = types.StringNull()
	}

	passed := true
	var reasons []string

	if !tc.ExpectedVerdict.IsNull() && !tc.ExpectedVerdict.IsUnknown() && strings.TrimSpace(tc.ExpectedVerdict.ValueString()) != "" {
		expV := strings.TrimSpace(tc.ExpectedVerdict.ValueString())
		out.ExpectedVerdict = types.StringValue(expV)
		if !strings.EqualFold(verdict, expV) {
			passed = false
			reasons = append(reasons, fmt.Sprintf("expected verdict %q but got %q", expV, verdict))
		}
	} else {
		out.ExpectedVerdict = types.StringNull()
	}

	if !tc.ExpectedAction.IsNull() && !tc.ExpectedAction.IsUnknown() && strings.TrimSpace(tc.ExpectedAction.ValueString()) != "" {
		expA := strings.TrimSpace(tc.ExpectedAction.ValueString())
		out.ExpectedAction = types.StringValue(expA)
		if !strings.EqualFold(actionStr, expA) {
			passed = false
			reasons = append(reasons, fmt.Sprintf("expected action %q but got %q", expA, actionStr))
		}
	} else {
		out.ExpectedAction = types.StringNull()
	}

	out.Passed = types.BoolValue(passed)
	if len(reasons) > 0 {
		out.FailureReason = types.StringValue(strings.Join(reasons, "; "))
	} else {
		out.FailureReason = types.StringNull()
	}

	return out, nil
}

// ===========================================================================
// Action: scm_dlp_dry_run_scan
// POST /v2/api/dry-run/scan
// Supports single-file or multi-file (test_cases) positive & negative testing
// ===========================================================================

type DryRunScanAction struct {
	client *dlpclient.Client
}

type DryRunScanActionModel struct {
	DataProfile         types.String          `tfsdk:"data_profile"`
	DataProfileId       types.String          `tfsdk:"data_profile_id"`
	FilePath            types.String          `tfsdk:"file_path"`
	FileContent         types.String          `tfsdk:"file_content"`
	FileName            types.String          `tfsdk:"file_name"`
	FileSha             types.String          `tfsdk:"file_sha"`
	ExpectedVerdict     types.String          `tfsdk:"expected_verdict"`
	ExpectedAction      types.String          `tfsdk:"expected_action"`
	FailOnMismatch      types.Bool            `tfsdk:"fail_on_mismatch"`
	TestCases           []DryRunTestCaseModel `tfsdk:"test_cases"`
	SourceDataProfileId types.String          `tfsdk:"source_data_profile_id"`
	DryRunServiceName   types.String          `tfsdk:"dry_run_service_name"`
}

func NewDryRunScanAction() action.Action {
	return &DryRunScanAction{}
}

func (a *DryRunScanAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_dry_run_scan"
}

func (a *DryRunScanAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "[Beta / Experimental] Runs one or more Enterprise DLP dry-run scans (positive and negative test cases) against a data profile using multipart form-data.",
		Attributes: map[string]schema.Attribute{
			"data_profile": schema.StringAttribute{
				Description: "JSON string representing the data profile configuration to test. Either data_profile or data_profile_id must be provided.",
				Optional:    true,
			},
			"data_profile_id": schema.StringAttribute{
				Description: "ID of an existing Enterprise DLP data profile (e.g. scm_dlp_data_profile.<name>.id) to fetch and test. Either data_profile or data_profile_id must be provided.",
				Optional:    true,
			},
			"file_path": schema.StringAttribute{
				Description: "Path to a single file to upload and scan. Used when test_cases is not specified.",
				Optional:    true,
			},
			"file_content": schema.StringAttribute{
				Description: "Raw content of a single file to scan. Used when test_cases is not specified.",
				Optional:    true,
			},
			"file_name": schema.StringAttribute{
				Description: "Name of the file being scanned (defaults to 'sample.txt' if file_content is used).",
				Optional:    true,
			},
			"file_sha": schema.StringAttribute{
				Description: "SHA256 hex string of the file (computed automatically if omitted).",
				Optional:    true,
			},
			"expected_verdict": schema.StringAttribute{
				Description: "Optional expected verdict for single-file scan (e.g. 'MATCHED' for positive tests or 'NOT_MATCHED' for negative tests).",
				Optional:    true,
			},
			"expected_action": schema.StringAttribute{
				Description: "Optional expected action for single-file scan (e.g. 'block', 'alert', 'allow').",
				Optional:    true,
			},
			"fail_on_mismatch": schema.BoolAttribute{
				Description: "Whether to fail the Terraform operation with an error diagnostic if any expected_verdict or expected_action does not match the scan result (defaults to true).",
				Optional:    true,
			},
			"test_cases": schema.ListNestedAttribute{
				Description: "Optional list of positive and negative test cases (files or inline content) to scan against the data profile.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Descriptive name for this test case (e.g. 'positive-pci-csv' or 'negative-benign-readme').",
							Optional:    true,
						},
						"file_path": schema.StringAttribute{
							Description: "Path to the local file to upload and scan for this test case.",
							Optional:    true,
						},
						"file_content": schema.StringAttribute{
							Description: "Inline content to scan for this test case.",
							Optional:    true,
						},
						"file_name": schema.StringAttribute{
							Description: "File name sent in the multipart upload (defaults to basename of file_path or 'sample.txt').",
							Optional:    true,
						},
						"file_sha": schema.StringAttribute{
							Description: "Optional SHA256 hex digest of the file.",
							Optional:    true,
						},
						"expected_verdict": schema.StringAttribute{
							Description: "Expected scan verdict: 'MATCHED' for positive test cases or 'NOT_MATCHED' for negative test cases.",
							Optional:    true,
						},
						"expected_action": schema.StringAttribute{
							Description: "Optional expected scan action (e.g. 'block', 'alert').",
							Optional:    true,
						},
					},
				},
			},
			"source_data_profile_id": schema.StringAttribute{
				Description: "Optional source data profile ID.",
				Optional:    true,
			},
			"dry_run_service_name": schema.StringAttribute{
				Description: "Optional dry-run service name (defaults to prisma-access).",
				Optional:    true,
			},
		},
	}
}

func (a *DryRunScanAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (a *DryRunScanAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data DryRunScanActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Invoking action scm_dlp_dry_run_scan")
	if resp != nil && resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: "Invoking action scm_dlp_dry_run_scan...",
		})
	}

	dataProfileStr, resolvedSourceID, err := resolveDryRunDataProfileJSON(ctx, a.client, data.DataProfile, data.DataProfileId)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Data Profile Input", err.Error())
		return
	}

	sourceID := resolvedSourceID
	if !data.SourceDataProfileId.IsNull() && !data.SourceDataProfileId.IsUnknown() && data.SourceDataProfileId.ValueString() != "" {
		sourceID = data.SourceDataProfileId.ValueString()
	}
	dryRunSvc := ""
	if !data.DryRunServiceName.IsNull() && !data.DryRunServiceName.IsUnknown() {
		dryRunSvc = data.DryRunServiceName.ValueString()
	}

	cases := data.TestCases
	if len(cases) == 0 {
		cases = []DryRunTestCaseModel{
			{
				Name:            types.StringValue("default"),
				FilePath:        data.FilePath,
				FileContent:     data.FileContent,
				FileName:        data.FileName,
				FileSha:         data.FileSha,
				ExpectedVerdict: data.ExpectedVerdict,
				ExpectedAction:  data.ExpectedAction,
			},
		}
	}

	failOnMismatch := true
	if !data.FailOnMismatch.IsNull() && !data.FailOnMismatch.IsUnknown() {
		failOnMismatch = data.FailOnMismatch.ValueBool()
	}

	var results []DryRunTestCaseResultModel
	var failures []string
	for _, tc := range cases {
		res, err := executeSingleDryRunCase(ctx, a.client, tc, dataProfileStr, sourceID, dryRunSvc)
		if err != nil {
			resp.Diagnostics.AddError("Dry-Run Scan Failed", err.Error())
			return
		}
		results = append(results, res)
		if !res.Passed.ValueBool() {
			failures = append(failures, fmt.Sprintf("%s (%s): %s", res.Name.ValueString(), res.FileName.ValueString(), res.FailureReason.ValueString()))
		}
	}

	if len(failures) > 0 && failOnMismatch {
		resp.Diagnostics.AddError(
			"Dry-Run Assertion Failed",
			fmt.Sprintf("%d of %d dry-run test case(s) failed expectation:\n- %s", len(failures), len(results), strings.Join(failures, "\n- ")),
		)
		return
	}

	completionMsg := fmt.Sprintf("Dry-run scan completed (%d test case(s), %d failure(s))", len(results), len(failures))
	tflog.Debug(ctx, completionMsg)
	if resp != nil && resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: completionMsg,
		})
	}
}

// ===========================================================================
// Data Source: scm_dlp_dry_run_scan
// POST /v2/api/dry-run/scan — Runs positive & negative dry-run test cases during plan/apply
// ===========================================================================

type DryRunScanDataSource struct {
	client *dlpclient.Client
}

type DryRunScanDataSourceModel struct {
	DataProfile         types.String                `tfsdk:"data_profile"`
	DataProfileId       types.String                `tfsdk:"data_profile_id"`
	FilePath            types.String                `tfsdk:"file_path"`
	FileContent         types.String                `tfsdk:"file_content"`
	FileName            types.String                `tfsdk:"file_name"`
	FileSha             types.String                `tfsdk:"file_sha"`
	ExpectedVerdict     types.String                `tfsdk:"expected_verdict"`
	ExpectedAction      types.String                `tfsdk:"expected_action"`
	FailOnMismatch      types.Bool                  `tfsdk:"fail_on_mismatch"`
	TestCases           []DryRunTestCaseModel       `tfsdk:"test_cases"`
	SourceDataProfileId types.String                `tfsdk:"source_data_profile_id"`
	DryRunServiceName   types.String                `tfsdk:"dry_run_service_name"`
	Tfid                types.String                `tfsdk:"tfid"`
	Status              types.String                `tfsdk:"status"`
	Verdict             types.String                `tfsdk:"verdict"`
	Action              types.String                `tfsdk:"action"`
	Passed              types.Bool                  `tfsdk:"passed"`
	ResultJson          types.String                `tfsdk:"result_json"`
	Results             []DryRunTestCaseResultModel `tfsdk:"results"`
}

func NewDryRunScanDataSource() datasource.DataSource {
	return &DryRunScanDataSource{}
}

func (d *DryRunScanDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_dry_run_scan"
}

func (d *DryRunScanDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		Description: "[Beta / Experimental] Executes one or more Enterprise DLP dry-run scans (positive and negative test cases) against a data profile and returns the scan results.",
		Attributes: map[string]dsschema.Attribute{
			"data_profile": dsschema.StringAttribute{
				Description: "JSON string representing the data profile configuration to test. Either data_profile or data_profile_id must be provided.",
				Optional:    true,
			},
			"data_profile_id": dsschema.StringAttribute{
				Description: "ID of an existing Enterprise DLP data profile (e.g. scm_dlp_data_profile.<name>.id) to fetch and test. Either data_profile or data_profile_id must be provided.",
				Optional:    true,
			},
			"file_path": dsschema.StringAttribute{
				Description: "Path to a single file to upload and scan. Used when test_cases is not specified.",
				Optional:    true,
			},
			"file_content": dsschema.StringAttribute{
				Description: "Raw content of a single file to scan. Used when test_cases is not specified.",
				Optional:    true,
			},
			"file_name": dsschema.StringAttribute{
				Description: "Name of the file being scanned (defaults to 'sample.txt' if file_content is used).",
				Optional:    true,
			},
			"file_sha": dsschema.StringAttribute{
				Description: "SHA256 hex string of the file (computed automatically if omitted).",
				Optional:    true,
				Computed:    true,
			},
			"expected_verdict": dsschema.StringAttribute{
				Description: "Optional expected verdict for single-file scan ('MATCHED' for positive test cases or 'NOT_MATCHED' for negative test cases).",
				Optional:    true,
			},
			"expected_action": dsschema.StringAttribute{
				Description: "Optional expected action for single-file scan (e.g. 'block', 'alert', 'allow').",
				Optional:    true,
			},
			"fail_on_mismatch": dsschema.BoolAttribute{
				Description: "Whether to fail the Terraform plan/apply if any expected_verdict or expected_action does not match the scan result (defaults to true).",
				Optional:    true,
			},
			"test_cases": dsschema.ListNestedAttribute{
				Description: "Optional list of positive and negative test cases (files or inline content) to scan against the data profile.",
				Optional:    true,
				NestedObject: dsschema.NestedAttributeObject{
					Attributes: map[string]dsschema.Attribute{
						"name": dsschema.StringAttribute{
							Description: "Descriptive name for this test case (e.g. 'positive-pci-csv' or 'negative-benign-readme').",
							Optional:    true,
						},
						"file_path": dsschema.StringAttribute{
							Description: "Path to the local file to upload and scan for this test case.",
							Optional:    true,
						},
						"file_content": dsschema.StringAttribute{
							Description: "Inline content to scan for this test case.",
							Optional:    true,
						},
						"file_name": dsschema.StringAttribute{
							Description: "File name sent in the multipart upload (defaults to basename of file_path or 'sample.txt').",
							Optional:    true,
						},
						"file_sha": dsschema.StringAttribute{
							Description: "Optional SHA256 hex digest of the file.",
							Optional:    true,
						},
						"expected_verdict": dsschema.StringAttribute{
							Description: "Expected scan verdict: 'MATCHED' for positive test cases or 'NOT_MATCHED' for negative test cases.",
							Optional:    true,
						},
						"expected_action": dsschema.StringAttribute{
							Description: "Optional expected scan action (e.g. 'block', 'alert').",
							Optional:    true,
						},
					},
				},
			},
			"source_data_profile_id": dsschema.StringAttribute{
				Description: "Optional source data profile ID.",
				Optional:    true,
			},
			"dry_run_service_name": dsschema.StringAttribute{
				Description: "Optional dry-run service name (defaults to prisma-access).",
				Optional:    true,
			},
			"tfid": dsschema.StringAttribute{
				Description: "Terraform ID.",
				Computed:    true,
			},
			"status": dsschema.StringAttribute{
				Description: "Dry-run scan status returned by the API (for single-file scan or first test case).",
				Computed:    true,
			},
			"verdict": dsschema.StringAttribute{
				Description: "Dry-run scan verdict returned by the API (for single-file scan or first test case).",
				Computed:    true,
			},
			"action": dsschema.StringAttribute{
				Description: "Dry-run scan action returned by the API (for single-file scan or first test case).",
				Computed:    true,
			},
			"passed": dsschema.BoolAttribute{
				Description: "True if all executed dry-run test cases matched their expected_verdict and expected_action.",
				Computed:    true,
			},
			"result_json": dsschema.StringAttribute{
				Description: "Full JSON response from POST /v2/api/dry-run/scan (for single-file scan or first test case).",
				Computed:    true,
			},
			"results": dsschema.ListNestedAttribute{
				Description: "Per-file dry-run scan results for all executed test cases.",
				Computed:    true,
				NestedObject: dsschema.NestedAttributeObject{
					Attributes: map[string]dsschema.Attribute{
						"name": dsschema.StringAttribute{
							Description: "Test case name.",
							Computed:    true,
						},
						"file_name": dsschema.StringAttribute{
							Description: "Scanned file name.",
							Computed:    true,
						},
						"file_sha": dsschema.StringAttribute{
							Description: "SHA256 hex digest of the scanned file.",
							Computed:    true,
						},
						"status": dsschema.StringAttribute{
							Description: "Dry-run scan status.",
							Computed:    true,
						},
						"verdict": dsschema.StringAttribute{
							Description: "Actual scan verdict ('MATCHED' or 'NOT_MATCHED').",
							Computed:    true,
						},
						"action": dsschema.StringAttribute{
							Description: "Actual policy action (e.g. 'block', 'alert').",
							Computed:    true,
						},
						"expected_verdict": dsschema.StringAttribute{
							Description: "Expected scan verdict if configured.",
							Computed:    true,
						},
						"expected_action": dsschema.StringAttribute{
							Description: "Expected scan action if configured.",
							Computed:    true,
						},
						"passed": dsschema.BoolAttribute{
							Description: "Whether this test case met its expected_verdict and expected_action.",
							Computed:    true,
						},
						"failure_reason": dsschema.StringAttribute{
							Description: "Mismatch explanation when passed is false.",
							Computed:    true,
						},
						"result_json": dsschema.StringAttribute{
							Description: "Raw JSON response from POST /v2/api/dry-run/scan for this test case.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *DryRunScanDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *DryRunScanDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DryRunScanDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runDryRunDataSource(ctx, d.client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func runDryRunDataSource(ctx context.Context, client *dlpclient.Client, data *DryRunScanDataSourceModel, diags *diag.Diagnostics) {
	dataProfileStr, resolvedSourceID, err := resolveDryRunDataProfileJSON(ctx, client, data.DataProfile, data.DataProfileId)
	if err != nil {
		diags.AddError("Invalid Data Profile Input", err.Error())
		return
	}

	sourceID := resolvedSourceID
	if !data.SourceDataProfileId.IsNull() && !data.SourceDataProfileId.IsUnknown() && data.SourceDataProfileId.ValueString() != "" {
		sourceID = data.SourceDataProfileId.ValueString()
	}
	dryRunSvc := ""
	if !data.DryRunServiceName.IsNull() && !data.DryRunServiceName.IsUnknown() {
		dryRunSvc = data.DryRunServiceName.ValueString()
	}

	cases := data.TestCases
	if len(cases) == 0 {
		cases = []DryRunTestCaseModel{
			{
				Name:            types.StringValue("default"),
				FilePath:        data.FilePath,
				FileContent:     data.FileContent,
				FileName:        data.FileName,
				FileSha:         data.FileSha,
				ExpectedVerdict: data.ExpectedVerdict,
				ExpectedAction:  data.ExpectedAction,
			},
		}
	}

	failOnMismatch := true
	if !data.FailOnMismatch.IsNull() && !data.FailOnMismatch.IsUnknown() {
		failOnMismatch = data.FailOnMismatch.ValueBool()
	}

	var results []DryRunTestCaseResultModel
	var failures []string
	hashAcc := sha256.New()

	for _, tc := range cases {
		res, err := executeSingleDryRunCase(ctx, client, tc, dataProfileStr, sourceID, dryRunSvc)
		if err != nil {
			diags.AddError("Dry-Run Scan Failed", err.Error())
			return
		}
		results = append(results, res)
		_, _ = hashAcc.Write([]byte(res.FileSha.ValueString()))
		if !res.Passed.ValueBool() {
			failures = append(failures, fmt.Sprintf("%s (%s): %s", res.Name.ValueString(), res.FileName.ValueString(), res.FailureReason.ValueString()))
		}
	}

	allPassed := len(failures) == 0
	data.Passed = types.BoolValue(allPassed)
	data.Results = results

	if len(results) == 1 {
		first := results[0]
		data.FileSha = first.FileSha
		data.Tfid = types.StringValue("dry_run:" + first.FileSha.ValueString())
		data.Status = first.Status
		data.Verdict = first.Verdict
		data.Action = first.Action
		data.ResultJson = first.ResultJson
	} else if len(results) > 1 {
		suiteSha := hex.EncodeToString(hashAcc.Sum(nil))
		first := results[0]
		data.FileSha = first.FileSha
		data.Tfid = types.StringValue("dry_run_suite:" + suiteSha[:16])
		data.Status = first.Status
		data.Verdict = first.Verdict
		data.Action = first.Action
		data.ResultJson = first.ResultJson
	}

	if !allPassed && failOnMismatch {
		diags.AddError(
			"Dry-Run Assertion Failed",
			fmt.Sprintf("%d of %d dry-run test case(s) failed expectation:\n- %s", len(failures), len(results), strings.Join(failures, "\n- ")),
		)
	}
}
