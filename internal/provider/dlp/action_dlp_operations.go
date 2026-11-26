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
	"time"

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
	_ action.Action                      = &ConfigExportAction{}
	_ action.ActionWithConfigure         = &ConfigExportAction{}
	_ action.Action                      = &ConfigImportAction{}
	_ action.ActionWithConfigure         = &ConfigImportAction{}
	_ action.Action                      = &DryRunScanAction{}
	_ action.ActionWithConfigure         = &DryRunScanAction{}
	_ datasource.DataSource              = &ConfigExportJobDataSource{}
	_ datasource.DataSourceWithConfigure = &ConfigExportJobDataSource{}
	_ datasource.DataSource              = &ConfigImportJobDataSource{}
	_ datasource.DataSourceWithConfigure = &ConfigImportJobDataSource{}
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
// Action: scm_dlp_config_export
// POST /v1/api/configuration/export
// Optional bounded polling via GET /v1/api/configuration/export/job/{id}
// and GET /v1/api/configuration/export/{id}
// ===========================================================================

type ConfigExportAction struct {
	client *dlpclient.Client
}

type ConfigExportActionModel struct {
	ExportType        types.String `tfsdk:"export_type"`
	WaitForCompletion types.Bool   `tfsdk:"wait_for_completion"`
	PollIntervalMs    types.Int64  `tfsdk:"poll_interval_ms"`
	TimeoutMs         types.Int64  `tfsdk:"timeout_ms"`
}

func NewConfigExportAction() action.Action {
	return &ConfigExportAction{}
}

func (a *ConfigExportAction) Metadata(_ context.Context, _ action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = "scm_dlp_config_export"
}

func (a *ConfigExportAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Triggers an asynchronous export of the tenant Enterprise DLP configuration (/v1/api/configuration/export).",
		Attributes: map[string]schema.Attribute{
			"export_type": schema.StringAttribute{
				Description: "Optional export type (e.g. JSON or ALL).",
				Optional:    true,
			},
			"wait_for_completion": schema.BoolAttribute{
				Description: "When true, polls GET /v1/api/configuration/export/job/{jobId} until a terminal status (completed, cancelled, failed) and retrieves the download link.",
				Optional:    true,
			},
			"poll_interval_ms": schema.Int64Attribute{
				Description: "Polling interval in milliseconds when wait_for_completion is true (default: 1000).",
				Optional:    true,
			},
			"timeout_ms": schema.Int64Attribute{
				Description: "Maximum polling timeout in milliseconds when wait_for_completion is true (default: 30000).",
				Optional:    true,
			},
		},
	}
}

func (a *ConfigExportAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (a *ConfigExportAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data ConfigExportActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Invoking action scm_dlp_config_export")
	if resp != nil && resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: "Invoking action scm_dlp_config_export...",
		})
	}

	var reqBody any
	if !data.ExportType.IsNull() && !data.ExportType.IsUnknown() && data.ExportType.ValueString() != "" {
		reqBody = map[string]any{
			"export_type": data.ExportType.ValueString(),
		}
	}

	var exportResp dlpclient.ConfigurationJobResponse
	if err := a.client.Post(ctx, "/v1/api/configuration/export", nil, reqBody, &exportResp); err != nil {
		resp.Diagnostics.AddError("Error Executing Config Export", err.Error())
		return
	}

	output := map[string]any{
		"export_job": exportResp,
	}

	if !data.WaitForCompletion.IsNull() && data.WaitForCompletion.ValueBool() && exportResp.JobID != "" {
		pollMs := int64(1000)
		if !data.PollIntervalMs.IsNull() && data.PollIntervalMs.ValueInt64() > 0 {
			pollMs = data.PollIntervalMs.ValueInt64()
		}
		if pollMs < 250 {
			pollMs = 250
		} else if pollMs > 5000 {
			pollMs = 5000
		}
		timeoutMs := int64(30000)
		if !data.TimeoutMs.IsNull() && data.TimeoutMs.ValueInt64() > 0 {
			timeoutMs = data.TimeoutMs.ValueInt64()
		}
		if timeoutMs < 1000 {
			timeoutMs = 1000
		} else if timeoutMs > 60000 {
			timeoutMs = 60000
		}

		deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
		var jobStatus dlpclient.ConfigurationJobResponse
		escapedJobID := dlpclient.EscapeID(exportResp.JobID)
		for {
			if err := a.client.Get(ctx, fmt.Sprintf("/v1/api/configuration/export/job/%s", escapedJobID), nil, &jobStatus); err != nil {
				resp.Diagnostics.AddError("Error Polling Config Export Job", err.Error())
				return
			}
			st := strings.ToLower(jobStatus.Status)
			if st == "completed" || st == "cancelled" || st == "failed" {
				break
			}
			if time.Now().After(deadline) {
				resp.Diagnostics.AddError("Config Export Polling Timeout", fmt.Sprintf("Export job %s did not reach terminal status within %dms", exportResp.JobID, timeoutMs))
				return
			}
			timer := time.NewTimer(time.Duration(pollMs) * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				resp.Diagnostics.AddError("Config Export Polling Cancelled", ctx.Err().Error())
				return
			case <-timer.C:
			}
		}
		output["completed_job"] = jobStatus
		if strings.EqualFold(jobStatus.Status, "completed") {
			var dlLink map[string]any
			if err := a.client.Get(ctx, fmt.Sprintf("/v1/api/configuration/export/%s", escapedJobID), nil, &dlLink); err == nil {
				output["download_link"] = dlLink
			}
		}
	}

	rawJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		rawJSON = []byte(fmt.Sprintf("%+v", output))
	}

	completionMsg := fmt.Sprintf("Config export job submitted successfully:\n%s", string(rawJSON))
	tflog.Debug(ctx, completionMsg)
	if resp != nil && resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: completionMsg,
		})
	}
}

// ===========================================================================
// Action: scm_dlp_config_import
// POST /v1/api/configuration/import (multipart field `file`)
// Optional conflict resolution via PUT /v1/api/configuration/import/job/{id}?action=...
// ===========================================================================

type ConfigImportAction struct {
	client *dlpclient.Client
}

type ConfigImportActionModel struct {
	FilePath       types.String `tfsdk:"file_path"`
	FileContent    types.String `tfsdk:"file_content"`
	FileName       types.String `tfsdk:"file_name"`
	ConflictAction types.String `tfsdk:"conflict_action"`
	ImportJobId    types.String `tfsdk:"import_job_id"`
}

func NewConfigImportAction() action.Action {
	return &ConfigImportAction{}
}

func (a *ConfigImportAction) Metadata(_ context.Context, _ action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = "scm_dlp_config_import"
}

func (a *ConfigImportAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Imports an Enterprise DLP tenant configuration bundle (/v1/api/configuration/import) or resolves conflicts on an existing import job (/v1/api/configuration/import/job/{id}?action=...).",
		Attributes: map[string]schema.Attribute{
			"file_path": schema.StringAttribute{
				Description: "Path to the configuration export file to upload. Either file_path, file_content, or (import_job_id + conflict_action) must be provided.",
				Optional:    true,
			},
			"file_content": schema.StringAttribute{
				Description: "Raw content of the configuration export file to upload.",
				Optional:    true,
			},
			"file_name": schema.StringAttribute{
				Description: "Filename to send in the multipart upload (defaults to basename of file_path or 'dlp-config-export.json').",
				Optional:    true,
			},
			"import_job_id": schema.StringAttribute{
				Description: "Existing import job ID when applying a conflict resolution action without uploading a new file.",
				Optional:    true,
			},
			"conflict_action": schema.StringAttribute{
				Description: "Optional conflict resolution action to apply via PUT /v1/api/configuration/import/job/{id}?action=... (OVERWRITE, REFRESH, CONTINUE, or CANCEL).",
				Optional:    true,
			},
		},
	}
}

func (a *ConfigImportAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (a *ConfigImportAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data ConfigImportActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Invoking action scm_dlp_config_import")
	if resp != nil && resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: "Invoking action scm_dlp_config_import...",
		})
	}

	jobID := ""
	if !data.ImportJobId.IsNull() && !data.ImportJobId.IsUnknown() {
		jobID = data.ImportJobId.ValueString()
	}

	output := map[string]any{}

	// Step 1: Upload configuration bundle if file_path or file_content is provided
	if (!data.FilePath.IsNull() && data.FilePath.ValueString() != "") || (!data.FileContent.IsNull() && data.FileContent.ValueString() != "") {
		var fileBytes []byte
		fileName := "dlp-config-export.json"
		if !data.FilePath.IsNull() && data.FilePath.ValueString() != "" {
			b, baseName, err := readValidatedUploadFile(data.FilePath.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("File Validation Error", err.Error())
				return
			}
			fileBytes = b
			fileName = baseName
		} else {
			fileBytes = []byte(data.FileContent.ValueString())
			if len(fileBytes) == 0 || len(fileBytes) > MaxUploadFileSize {
				resp.Diagnostics.AddError("File Validation Error", "file_content must be non-empty and at most 20MB.")
				return
			}
		}
		if !data.FileName.IsNull() && data.FileName.ValueString() != "" {
			fileName = filepath.Base(data.FileName.ValueString())
		}

		parts := []dlpclient.MultipartPart{
			{
				FieldName:   "file",
				FileName:    fileName,
				ContentType: "application/octet-stream",
				Data:        fileBytes,
			},
		}
		var importResp map[string]any
		if err := a.client.DoMultipart(ctx, "POST", "/v1/api/configuration/import", nil, parts, &importResp); err != nil {
			resp.Diagnostics.AddError("Error Executing Config Import", err.Error())
			return
		}
		output["import_job"] = importResp
		if jid, ok := importResp["job_id"].(string); ok && jid != "" {
			jobID = jid
		}
	} else if jobID == "" {
		resp.Diagnostics.AddError("Missing Input", "Provide 'file_path', 'file_content', or 'import_job_id' with 'conflict_action'.")
		return
	}

	// Step 2: Optional conflict resolution action (OVERWRITE, REFRESH, CONTINUE, CANCEL)
	if !data.ConflictAction.IsNull() && data.ConflictAction.ValueString() != "" {
		act := strings.ToUpper(data.ConflictAction.ValueString())
		validActions := map[string]bool{"OVERWRITE": true, "REFRESH": true, "CONTINUE": true, "CANCEL": true}
		if !validActions[act] {
			resp.Diagnostics.AddError("Invalid Conflict Action", "conflict_action must be one of OVERWRITE, REFRESH, CONTINUE, or CANCEL.")
			return
		}
		if jobID == "" {
			resp.Diagnostics.AddError("Missing Import Job ID", "Cannot apply conflict_action without an import_job_id.")
			return
		}
		var conflictResp map[string]any
		if err := a.client.Put(ctx, fmt.Sprintf("/v1/api/configuration/import/job/%s", dlpclient.EscapeID(jobID)), map[string]string{"action": act}, nil, &conflictResp); err != nil {
			resp.Diagnostics.AddError("Error Resolving Config Import Conflicts", err.Error())
			return
		}
		output["conflict_resolution"] = conflictResp
	}

	rawJSON, _ := json.MarshalIndent(output, "", "  ")
	completionMsg := fmt.Sprintf("Config import operation completed:\n%s", string(rawJSON))
	tflog.Debug(ctx, completionMsg)
	if resp != nil && resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: completionMsg,
		})
	}
}

// ===========================================================================
// Data Source: scm_dlp_config_export_job
// GET /v1/api/configuration/export/job/{exportJobId}
// GET /v1/api/configuration/export/{exportJobId} (when completed)
// ===========================================================================

type ConfigExportJobDataSource struct {
	client *dlpclient.Client
}

type ConfigExportJobDataSourceModel struct {
	ExportJobId types.String `tfsdk:"export_job_id"`
	Tfid        types.String `tfsdk:"tfid"`
	Status      types.String `tfsdk:"status"`
	ExportType  types.String `tfsdk:"export_type"`
	DownloadUrl types.String `tfsdk:"download_url"`
	ResultJson  types.String `tfsdk:"result_json"`
}

func NewConfigExportJobDataSource() datasource.DataSource {
	return &ConfigExportJobDataSource{}
}

func (d *ConfigExportJobDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_config_export_job"
}

func (d *ConfigExportJobDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		Description: "Reads the status and download URL of an Enterprise DLP configuration export job (/v1/api/configuration/export/job/{id}).",
		Attributes: map[string]dsschema.Attribute{
			"export_job_id": dsschema.StringAttribute{
				Description: "Configuration export job ID.",
				Required:    true,
			},
			"tfid": dsschema.StringAttribute{
				Description: "Terraform ID.",
				Computed:    true,
			},
			"status": dsschema.StringAttribute{
				Description: "Export job status (e.g. completed, cancelled, failed, in_progress).",
				Computed:    true,
			},
			"export_type": dsschema.StringAttribute{
				Description: "Export job type.",
				Computed:    true,
			},
			"download_url": dsschema.StringAttribute{
				Description: "Download URL when the export job is completed.",
				Computed:    true,
			},
			"result_json": dsschema.StringAttribute{
				Description: "Raw JSON response from GET /v1/api/configuration/export/job/{id}.",
				Computed:    true,
			},
		},
	}
}

func (d *ConfigExportJobDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *ConfigExportJobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConfigExportJobDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobID := data.ExportJobId.ValueString()
	var jobResp map[string]any
	if err := d.client.Get(ctx, fmt.Sprintf("/v1/api/configuration/export/job/%s", dlpclient.EscapeID(jobID)), nil, &jobResp); err != nil {
		resp.Diagnostics.AddError("Error Reading Config Export Job", err.Error())
		return
	}

	rawJSON, _ := json.Marshal(jobResp)
	data.Tfid = types.StringValue("export_job:" + jobID)
	data.ResultJson = types.StringValue(string(rawJSON))
	if st, ok := jobResp["status"].(string); ok {
		data.Status = types.StringValue(st)
	} else {
		data.Status = types.StringNull()
	}
	if et, ok := jobResp["export_type"].(string); ok {
		data.ExportType = types.StringValue(et)
	} else {
		data.ExportType = types.StringNull()
	}
	if u, ok := jobResp["url"].(string); ok && u != "" {
		data.DownloadUrl = types.StringValue(u)
	} else {
		data.DownloadUrl = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ===========================================================================
// Data Source: scm_dlp_config_import_job
// GET /v1/api/configuration/import/job/{importJobId}
// ===========================================================================

type ConfigImportJobDataSource struct {
	client *dlpclient.Client
}

type ConfigImportJobDataSourceModel struct {
	ImportJobId types.String `tfsdk:"import_job_id"`
	Tfid        types.String `tfsdk:"tfid"`
	Status      types.String `tfsdk:"status"`
	ResultJson  types.String `tfsdk:"result_json"`
}

func NewConfigImportJobDataSource() datasource.DataSource {
	return &ConfigImportJobDataSource{}
}

func (d *ConfigImportJobDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_config_import_job"
}

func (d *ConfigImportJobDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		Description: "Reads the status and conflict details of an Enterprise DLP configuration import job (/v1/api/configuration/import/job/{id}).",
		Attributes: map[string]dsschema.Attribute{
			"import_job_id": dsschema.StringAttribute{
				Description: "Configuration import job ID.",
				Required:    true,
			},
			"tfid": dsschema.StringAttribute{
				Description: "Terraform ID.",
				Computed:    true,
			},
			"status": dsschema.StringAttribute{
				Description: "Import job status.",
				Computed:    true,
			},
			"result_json": dsschema.StringAttribute{
				Description: "Raw JSON response from GET /v1/api/configuration/import/job/{id}.",
				Computed:    true,
			},
		},
	}
}

func (d *ConfigImportJobDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *ConfigImportJobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConfigImportJobDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobID := data.ImportJobId.ValueString()
	var jobResp map[string]any
	if err := d.client.Get(ctx, fmt.Sprintf("/v1/api/configuration/import/job/%s", dlpclient.EscapeID(jobID)), nil, &jobResp); err != nil {
		resp.Diagnostics.AddError("Error Reading Config Import Job", err.Error())
		return
	}

	rawJSON, _ := json.Marshal(jobResp)
	data.Tfid = types.StringValue("import_job:" + jobID)
	data.ResultJson = types.StringValue(string(rawJSON))
	if st, ok := jobResp["status"].(string); ok {
		data.Status = types.StringValue(st)
	} else {
		data.Status = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

func (a *DryRunScanAction) Metadata(_ context.Context, _ action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = "scm_dlp_dry_run_scan"
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

func (d *DryRunScanDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_dry_run_scan"
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
