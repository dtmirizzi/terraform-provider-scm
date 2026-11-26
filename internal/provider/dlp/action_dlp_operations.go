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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/paloaltonetworks/terraform-provider-scm/internal/dlpclient"
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

// ===========================================================================
// Action: scm_dlp_dry_run_scan
// POST /v2/api/dry-run/scan
// Uses DoMultipart with fileSha, dataProfile JSON, and file content
// ===========================================================================

type DryRunScanAction struct {
	client *dlpclient.Client
}

type DryRunScanActionModel struct {
	DataProfile         types.String `tfsdk:"data_profile"`
	FilePath            types.String `tfsdk:"file_path"`
	FileContent         types.String `tfsdk:"file_content"`
	FileName            types.String `tfsdk:"file_name"`
	FileSha             types.String `tfsdk:"file_sha"`
	SourceDataProfileId types.String `tfsdk:"source_data_profile_id"`
	DryRunServiceName   types.String `tfsdk:"dry_run_service_name"`
}

func NewDryRunScanAction() action.Action {
	return &DryRunScanAction{}
}

func (a *DryRunScanAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_dry_run_scan"
}

func (a *DryRunScanAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Runs an Enterprise DLP dry-run scan against a data profile using multipart form-data.",
		Attributes: map[string]schema.Attribute{
			"data_profile": schema.StringAttribute{
				Description: "JSON string representing the data profile configuration to test.",
				Required:    true,
			},
			"file_path": schema.StringAttribute{
				Description: "Path to the file to upload and scan. Either file_path or file_content must be provided.",
				Optional:    true,
			},
			"file_content": schema.StringAttribute{
				Description: "Raw content of the file to scan. Either file_path or file_content must be provided.",
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

	var fileBytes []byte
	fileName := "sample.txt"

	if !data.FilePath.IsNull() && data.FilePath.ValueString() != "" {
		b, baseName, err := readValidatedUploadFile(data.FilePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("File Validation Error", err.Error())
			return
		}
		fileBytes = b
		fileName = baseName
	} else if !data.FileContent.IsNull() {
		fileBytes = []byte(data.FileContent.ValueString())
		if len(fileBytes) == 0 || len(fileBytes) > MaxUploadFileSize {
			resp.Diagnostics.AddError("File Validation Error", "file_content must be non-empty and at most 20MB.")
			return
		}
	} else {
		resp.Diagnostics.AddError("Missing Input", "Either 'file_path' or 'file_content' must be provided.")
		return
	}

	if !data.FileName.IsNull() && data.FileName.ValueString() != "" {
		fileName = filepath.Base(data.FileName.ValueString())
	}

	fileSha := ""
	if !data.FileSha.IsNull() && data.FileSha.ValueString() != "" {
		fileSha = data.FileSha.ValueString()
	} else {
		hash := sha256.Sum256(fileBytes)
		fileSha = hex.EncodeToString(hash[:])
	}

	dataProfileStr, err := normalizeDryRunDataProfileJSON(data.DataProfile.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Data Profile JSON", "The 'data_profile' argument must be valid JSON.")
		return
	}

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
	if !data.SourceDataProfileId.IsNull() && data.SourceDataProfileId.ValueString() != "" {
		parts = append(parts, dlpclient.MultipartPart{
			FieldName: "sourceDataProfileId",
			Data:      []byte(data.SourceDataProfileId.ValueString()),
		})
	}
	if !data.DryRunServiceName.IsNull() && data.DryRunServiceName.ValueString() != "" {
		parts = append(parts, dlpclient.MultipartPart{
			FieldName: "dryRunServiceName",
			Data:      []byte(data.DryRunServiceName.ValueString()),
		})
	}

	var scanResult map[string]any
	if err := a.client.DoMultipart(ctx, "POST", "/v2/api/dry-run/scan", nil, parts, &scanResult); err != nil {
		resp.Diagnostics.AddError("Dry-Run Scan Failed", err.Error())
		return
	}

	rawJSON, err := json.MarshalIndent(scanResult, "", "  ")
	if err != nil {
		rawJSON = []byte(fmt.Sprintf("%+v", scanResult))
	}

	completionMsg := fmt.Sprintf("Dry-run scan completed successfully:\n%s", string(rawJSON))
	tflog.Debug(ctx, completionMsg)
	if resp != nil && resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: completionMsg,
		})
	}
}

// ===========================================================================
// Data Source: scm_dlp_dry_run_scan
// POST /v2/api/dry-run/scan — Runs a dry-run scan during plan/apply and exposes result_json
// ===========================================================================

type DryRunScanDataSource struct {
	client *dlpclient.Client
}

type DryRunScanDataSourceModel struct {
	DataProfile         types.String `tfsdk:"data_profile"`
	FilePath            types.String `tfsdk:"file_path"`
	FileContent         types.String `tfsdk:"file_content"`
	FileName            types.String `tfsdk:"file_name"`
	FileSha             types.String `tfsdk:"file_sha"`
	SourceDataProfileId types.String `tfsdk:"source_data_profile_id"`
	DryRunServiceName   types.String `tfsdk:"dry_run_service_name"`
	Tfid                types.String `tfsdk:"tfid"`
	Status              types.String `tfsdk:"status"`
	Verdict             types.String `tfsdk:"verdict"`
	ResultJson          types.String `tfsdk:"result_json"`
}

func NewDryRunScanDataSource() datasource.DataSource {
	return &DryRunScanDataSource{}
}

func (d *DryRunScanDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_dry_run_scan"
}

func (d *DryRunScanDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		Description: "Executes an Enterprise DLP dry-run scan against a data profile and returns the scan result.",
		Attributes: map[string]dsschema.Attribute{
			"data_profile": dsschema.StringAttribute{
				Description: "JSON string representing the data profile configuration to test.",
				Required:    true,
			},
			"file_path": dsschema.StringAttribute{
				Description: "Path to the file to upload and scan. Either file_path or file_content must be provided.",
				Optional:    true,
			},
			"file_content": dsschema.StringAttribute{
				Description: "Raw content of the file to scan. Either file_path or file_content must be provided.",
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
				Description: "Dry-run scan status returned by the API.",
				Computed:    true,
			},
			"verdict": dsschema.StringAttribute{
				Description: "Dry-run scan verdict returned by the API.",
				Computed:    true,
			},
			"result_json": dsschema.StringAttribute{
				Description: "Full JSON response from POST /v2/api/dry-run/scan.",
				Computed:    true,
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

	var fileBytes []byte
	fileName := "sample.txt"

	if !data.FilePath.IsNull() && data.FilePath.ValueString() != "" {
		b, baseName, err := readValidatedUploadFile(data.FilePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("File Validation Error", err.Error())
			return
		}
		fileBytes = b
		fileName = baseName
	} else if !data.FileContent.IsNull() {
		fileBytes = []byte(data.FileContent.ValueString())
		if len(fileBytes) == 0 || len(fileBytes) > MaxUploadFileSize {
			resp.Diagnostics.AddError("File Validation Error", "file_content must be non-empty and at most 20MB.")
			return
		}
	} else {
		resp.Diagnostics.AddError("Missing Input", "Either 'file_path' or 'file_content' must be provided.")
		return
	}

	if !data.FileName.IsNull() && data.FileName.ValueString() != "" {
		fileName = filepath.Base(data.FileName.ValueString())
	}

	fileSha := ""
	if !data.FileSha.IsNull() && data.FileSha.ValueString() != "" {
		fileSha = data.FileSha.ValueString()
	} else {
		hash := sha256.Sum256(fileBytes)
		fileSha = hex.EncodeToString(hash[:])
	}

	dataProfileStr, err := normalizeDryRunDataProfileJSON(data.DataProfile.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Data Profile JSON", "The 'data_profile' argument must be valid JSON.")
		return
	}

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
	if !data.SourceDataProfileId.IsNull() && data.SourceDataProfileId.ValueString() != "" {
		parts = append(parts, dlpclient.MultipartPart{
			FieldName: "sourceDataProfileId",
			Data:      []byte(data.SourceDataProfileId.ValueString()),
		})
	}
	if !data.DryRunServiceName.IsNull() && data.DryRunServiceName.ValueString() != "" {
		parts = append(parts, dlpclient.MultipartPart{
			FieldName: "dryRunServiceName",
			Data:      []byte(data.DryRunServiceName.ValueString()),
		})
	}

	var scanResult map[string]any
	if err := d.client.DoMultipart(ctx, "POST", "/v2/api/dry-run/scan", nil, parts, &scanResult); err != nil {
		resp.Diagnostics.AddError("Dry-Run Scan Failed", err.Error())
		return
	}

	rawJSON, _ := json.Marshal(scanResult)
	data.FileSha = types.StringValue(fileSha)
	data.Tfid = types.StringValue("dry_run:" + fileSha)
	data.ResultJson = types.StringValue(string(rawJSON))
	if st, ok := scanResult["status"].(string); ok {
		data.Status = types.StringValue(st)
	} else {
		data.Status = types.StringNull()
	}
	if vd, ok := scanResult["verdict"].(string); ok {
		data.Verdict = types.StringValue(vd)
	} else if rpt, ok := scanResult["report"].(map[string]any); ok {
		if vd, ok := rpt["data_pattern_rule_1_verdict"].(string); ok {
			data.Verdict = types.StringValue(vd)
		} else {
			data.Verdict = types.StringNull()
		}
	} else {
		data.Verdict = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
