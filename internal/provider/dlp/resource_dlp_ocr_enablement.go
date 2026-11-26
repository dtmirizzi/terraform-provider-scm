package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
	"github.com/paloaltonetworks/terraform-provider-scm/internal/provider/dlp/client"
)

// ============================================================================
// RESOURCE: scm_dlp_ocr_enablement
// ============================================================================

var (
	_ resource.Resource                = &dlpOcrEnablementResource{}
	_ resource.ResourceWithConfigure   = &dlpOcrEnablementResource{}
	_ resource.ResourceWithImportState = &dlpOcrEnablementResource{}
)

func NewDlpOcrEnablementResource() resource.Resource {
	return &dlpOcrEnablementResource{}
}

type dlpOcrEnablementResource struct {
	client *dlpclient.Client
}

func (r *dlpOcrEnablementResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_ocr_enablement"
}

func (r *dlpOcrEnablementResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.DlpOcrEnablementResourceSchema
}

func (r *dlpOcrEnablementResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (r *dlpOcrEnablementResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.DlpOcrEnablementModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := dlpclient.OcrEnablementDTO{
		TargetServiceName: plan.TargetServiceName.ValueString(),
		OcrEnablement:     plan.OcrEnablement.ValueString(),
	}

	var raw json.RawMessage
	if err := r.client.Put(ctx, "/v2/api/ocr", nil, dto, &raw); err != nil {
		resp.Diagnostics.AddError("Error setting OCR enablement", fmt.Sprintf("Failed to call PUT /v2/api/ocr: %s", err))
		return
	}
	putResp, _ := parseOcrResponse(raw, plan.TargetServiceName.ValueString())

	state := plan
	if putResp.TenantID != nil {
		state.TenantId = types.StringValue(*putResp.TenantID)
	} else if state.TenantId.IsNull() || state.TenantId.IsUnknown() {
		state.TenantId = types.StringValue("ocr:" + plan.TargetServiceName.ValueString())
	}
	state.Tfid = types.StringValue(plan.TargetServiceName.ValueString())
	if putResp.OcrEnablement != "" {
		state.OcrEnablement = types.StringValue(putResp.OcrEnablement)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dlpOcrEnablementResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.DlpOcrEnablementModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetSvc := state.TargetServiceName.ValueString()
	if targetSvc == "" && !state.Tfid.IsNull() {
		targetSvc = state.Tfid.ValueString()
	}

	var raw json.RawMessage
	params := map[string]string{"target-service-name": targetSvc}
	if err := r.client.Get(ctx, "/v2/api/ocr", params, &raw); err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading OCR enablement", fmt.Sprintf("Failed to call GET /v2/api/ocr: %s", err))
		return
	}
	getResp, found := parseOcrResponse(raw, targetSvc)
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	state.TargetServiceName = types.StringValue(targetSvc)
	if getResp.OcrEnablement != "" {
		state.OcrEnablement = types.StringValue(getResp.OcrEnablement)
	}
	if getResp.TenantID != nil {
		state.TenantId = types.StringValue(*getResp.TenantID)
	} else if state.TenantId.IsNull() || state.TenantId.IsUnknown() {
		state.TenantId = types.StringValue("ocr:" + targetSvc)
	}
	state.Tfid = types.StringValue(targetSvc)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dlpOcrEnablementResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.DlpOcrEnablementModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := dlpclient.OcrEnablementDTO{
		TargetServiceName: plan.TargetServiceName.ValueString(),
		OcrEnablement:     plan.OcrEnablement.ValueString(),
	}

	var raw json.RawMessage
	if err := r.client.Put(ctx, "/v2/api/ocr", nil, dto, &raw); err != nil {
		resp.Diagnostics.AddError("Error updating OCR enablement", fmt.Sprintf("Failed to call PUT /v2/api/ocr: %s", err))
		return
	}
	putResp, _ := parseOcrResponse(raw, plan.TargetServiceName.ValueString())

	state := plan
	if putResp.TenantID != nil {
		state.TenantId = types.StringValue(*putResp.TenantID)
	} else if state.TenantId.IsNull() || state.TenantId.IsUnknown() {
		state.TenantId = types.StringValue("ocr:" + plan.TargetServiceName.ValueString())
	}
	state.Tfid = types.StringValue(plan.TargetServiceName.ValueString())
	if putResp.OcrEnablement != "" {
		state.OcrEnablement = types.StringValue(putResp.OcrEnablement)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dlpOcrEnablementResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.DlpOcrEnablementModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetSvc := state.TargetServiceName.ValueString()
	if targetSvc != "" {
		dto := dlpclient.OcrEnablementDTO{
			TargetServiceName: targetSvc,
			OcrEnablement:     "DISABLED",
		}
		var raw json.RawMessage
		if err := r.client.Put(ctx, "/v2/api/ocr", nil, dto, &raw); err != nil {
			if !dlpclient.IsNotFound(err) {
				resp.Diagnostics.AddError("Error disabling OCR enablement on delete", fmt.Sprintf("Failed to call PUT /v2/api/ocr: %s", err))
				return
			}
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *dlpOcrEnablementResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by target_service_name
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target_service_name"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tfid"), req.ID)...)
}

// ============================================================================
// DATA SOURCE: scm_dlp_ocr_enablement
// ============================================================================

var (
	_ datasource.DataSource              = &dlpOcrEnablementDataSource{}
	_ datasource.DataSourceWithConfigure = &dlpOcrEnablementDataSource{}
)

func NewDlpOcrEnablementDataSource() datasource.DataSource {
	return &dlpOcrEnablementDataSource{}
}

type dlpOcrEnablementDataSource struct {
	client *dlpclient.Client
}

func (d *dlpOcrEnablementDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_ocr_enablement"
}

func (d *dlpOcrEnablementDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DlpOcrEnablementDataSourceSchema
}

func (d *dlpOcrEnablementDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *dlpOcrEnablementDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.DlpOcrEnablementModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetSvc := config.TargetServiceName.ValueString()
	var raw json.RawMessage
	params := map[string]string{"target-service-name": targetSvc}
	if err := d.client.Get(ctx, "/v2/api/ocr", params, &raw); err != nil {
		resp.Diagnostics.AddError("Error reading OCR enablement", fmt.Sprintf("Failed to call GET /v2/api/ocr: %s", err))
		return
	}
	getResp, found := parseOcrResponse(raw, targetSvc)
	if !found {
		resp.Diagnostics.AddError("OCR Enablement Not Found", fmt.Sprintf("No OCR enablement found for target service %q", targetSvc))
		return
	}

	state := config
	state.TargetServiceName = types.StringValue(targetSvc)
	if getResp.OcrEnablement != "" {
		state.OcrEnablement = types.StringValue(getResp.OcrEnablement)
	}
	if getResp.TenantID != nil {
		state.TenantId = types.StringValue(*getResp.TenantID)
	} else {
		state.TenantId = types.StringValue("ocr:" + targetSvc)
	}
	state.Tfid = types.StringValue(targetSvc)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func parseOcrResponse(raw json.RawMessage, targetSvc string) (dlpclient.OcrEnablementDTO, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return dlpclient.OcrEnablementDTO{TargetServiceName: targetSvc}, targetSvc == ""
	}
	if trimmed[0] == '[' {
		var list []dlpclient.OcrEnablementDTO
		if err := json.Unmarshal(trimmed, &list); err == nil {
			for _, item := range list {
				if strings.EqualFold(item.TargetServiceName, targetSvc) || strings.EqualFold(item.ServiceName, targetSvc) {
					return item, true
				}
			}
			return dlpclient.OcrEnablementDTO{}, false
		}
	}
	var single dlpclient.OcrEnablementDTO
	if err := json.Unmarshal(trimmed, &single); err == nil {
		if targetSvc == "" || strings.EqualFold(single.TargetServiceName, targetSvc) || strings.EqualFold(single.ServiceName, targetSvc) {
			return single, true
		}
		return dlpclient.OcrEnablementDTO{}, false
	}
	return single, false
}
