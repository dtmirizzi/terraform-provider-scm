package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/paloaltonetworks/terraform-provider-scm/internal/dlpclient"
	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
)

// ============================================================================
// RESOURCE: scm_dlp_alert_settings
// ============================================================================

var (
	_ resource.Resource                = &dlpAlertSettingsResource{}
	_ resource.ResourceWithConfigure   = &dlpAlertSettingsResource{}
	_ resource.ResourceWithImportState = &dlpAlertSettingsResource{}
)

func NewDlpAlertSettingsResource() resource.Resource {
	return &dlpAlertSettingsResource{}
}

type dlpAlertSettingsResource struct {
	client *dlpclient.Client
}

func (r *dlpAlertSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "scm_dlp_alert_settings"
}

func (r *dlpAlertSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.DlpAlertSettingsResourceSchema
}

func (r *dlpAlertSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (r *dlpAlertSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.DlpAlertSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := dlpAlertSettingsModelToDTO(&plan)
	var putResp dlpclient.AlertSettingsDTO
	if err := r.client.Put(ctx, "/v1/api/alerts/setting", nil, dto, &putResp); err != nil {
		resp.Diagnostics.AddError("Error setting alert settings", fmt.Sprintf("Failed to call PUT /v1/api/alerts/setting: %s", err))
		return
	}

	state := plan
	dlpAlertSettingsMergeDTO(&state, &putResp)
	if state.Id.IsNull() || state.Id.ValueString() == "" {
		state.Id = types.StringValue("alert_settings")
	}
	state.Tfid = types.StringValue(state.Id.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dlpAlertSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.DlpAlertSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var getResp dlpclient.AlertSettingsDTO
	if err := r.client.Get(ctx, "/v1/api/alerts/setting", nil, &getResp); err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading alert settings", fmt.Sprintf("Failed to call GET /v1/api/alerts/setting: %s", err))
		return
	}

	dlpAlertSettingsMergeDTO(&state, &getResp)
	if state.Id.IsNull() || state.Id.ValueString() == "" {
		state.Id = types.StringValue("alert_settings")
	}
	state.Tfid = types.StringValue(state.Id.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dlpAlertSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.DlpAlertSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := dlpAlertSettingsModelToDTO(&plan)
	var putResp dlpclient.AlertSettingsDTO
	if err := r.client.Put(ctx, "/v1/api/alerts/setting", nil, dto, &putResp); err != nil {
		resp.Diagnostics.AddError("Error updating alert settings", fmt.Sprintf("Failed to call PUT /v1/api/alerts/setting: %s", err))
		return
	}

	state := plan
	dlpAlertSettingsMergeDTO(&state, &putResp)
	if state.Id.IsNull() || state.Id.ValueString() == "" {
		state.Id = types.StringValue("alert_settings")
	}
	state.Tfid = types.StringValue(state.Id.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dlpAlertSettingsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting scm_dlp_alert_settings from Terraform state (singleton configuration)")
	resp.State.RemoveResource(ctx)
}

func (r *dlpAlertSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ============================================================================
// DATA SOURCE: scm_dlp_alert_settings
// ============================================================================

var (
	_ datasource.DataSource              = &dlpAlertSettingsDataSource{}
	_ datasource.DataSourceWithConfigure = &dlpAlertSettingsDataSource{}
)

func NewDlpAlertSettingsDataSource() datasource.DataSource {
	return &dlpAlertSettingsDataSource{}
}

type dlpAlertSettingsDataSource struct {
	client *dlpclient.Client
}

func (d *dlpAlertSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_alert_settings"
}

func (d *dlpAlertSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DlpAlertSettingsDataSourceSchema
}

func (d *dlpAlertSettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *dlpAlertSettingsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getResp dlpclient.AlertSettingsDTO
	if err := d.client.Get(ctx, "/v1/api/alerts/setting", nil, &getResp); err != nil {
		resp.Diagnostics.AddError("Error reading alert settings", fmt.Sprintf("Failed to call GET /v1/api/alerts/setting: %s", err))
		return
	}

	var state models.DlpAlertSettingsModel
	dlpAlertSettingsMergeDTO(&state, &getResp)
	if state.Id.IsNull() || state.Id.ValueString() == "" {
		state.Id = types.StringValue("alert_settings")
	}
	state.Tfid = types.StringValue(state.Id.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func dlpAlertSettingsModelToDTO(m *models.DlpAlertSettingsModel) *dlpclient.AlertSettingsDTO {
	dto := &dlpclient.AlertSettingsDTO{}
	if !m.IncidentNotificationEnablement.IsNull() && !m.IncidentNotificationEnablement.IsUnknown() {
		v := m.IncidentNotificationEnablement.ValueBool()
		dto.IncidentNotificationEnablement = &v
	}
	if !m.DetectionExemptionPeriodMinutes.IsNull() && !m.DetectionExemptionPeriodMinutes.IsUnknown() {
		v := m.DetectionExemptionPeriodMinutes.ValueInt64()
		dto.DetectionExemptionPeriodMinutes = &v
	}
	return dto
}

func dlpAlertSettingsMergeDTO(m *models.DlpAlertSettingsModel, dto *dlpclient.AlertSettingsDTO) {
	if dto == nil {
		return
	}
	if dto.IncidentNotificationEnablement != nil {
		m.IncidentNotificationEnablement = types.BoolValue(*dto.IncidentNotificationEnablement)
	}
	if dto.DetectionExemptionPeriodMinutes != nil {
		m.DetectionExemptionPeriodMinutes = types.Int64Value(*dto.DetectionExemptionPeriodMinutes)
	}
}

// ============================================================================
// DATA SOURCE: scm_dlp_tenant_channels
// ============================================================================

var (
	_ datasource.DataSource              = &dlpTenantChannelsDataSource{}
	_ datasource.DataSourceWithConfigure = &dlpTenantChannelsDataSource{}
)

func NewDlpTenantChannelsDataSource() datasource.DataSource {
	return &dlpTenantChannelsDataSource{}
}

type dlpTenantChannelsDataSource struct {
	client *dlpclient.Client
}

func (d *dlpTenantChannelsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_tenant_channels"
}

func (d *dlpTenantChannelsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DlpTenantChannelsDataSourceSchema
}

func (d *dlpTenantChannelsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *dlpTenantChannelsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var rawMap map[string]any
	if err := d.client.Get(ctx, "/v1/api/channels", nil, &rawMap); err != nil {
		resp.Diagnostics.AddError("Error reading tenant channels", fmt.Sprintf("Failed to call GET /v1/api/channels: %s", err))
		return
	}

	elements := make(map[string]attr.Value)
	for k, v := range rawMap {
		switch val := v.(type) {
		case bool:
			elements[k] = types.BoolValue(val)
		case string:
			elements[k] = types.BoolValue(val == "true" || val == "1" || val == "enabled" || val == "ENABLED")
		default:
			// Fallback boolean conversion if truthy
			elements[k] = types.BoolValue(val != nil && val != false && val != 0)
		}
	}

	mapVal, diags := types.MapValue(types.BoolType, elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := models.DlpTenantChannelsModel{
		Tfid:     types.StringValue("tenant_channels"),
		Channels: mapVal,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
