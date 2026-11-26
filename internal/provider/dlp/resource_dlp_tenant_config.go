package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/paloaltonetworks/terraform-provider-scm/internal/dlpclient"
	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
)

// ============================================================================
// RESOURCE: scm_dlp_tenant_config
// ============================================================================

var (
	_ resource.Resource                = &dlpTenantConfigResource{}
	_ resource.ResourceWithConfigure   = &dlpTenantConfigResource{}
	_ resource.ResourceWithImportState = &dlpTenantConfigResource{}
)

func NewDlpTenantConfigResource() resource.Resource {
	return &dlpTenantConfigResource{}
}

type dlpTenantConfigResource struct {
	client *dlpclient.Client
}

func (r *dlpTenantConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_tenant_config"
}

func (r *dlpTenantConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.DlpTenantConfigResourceSchema
}

func (r *dlpTenantConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (r *dlpTenantConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.DlpTenantConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := dlpTenantConfigModelToDTO(&plan)

	errMsgs, err := dlpclient.ValidateTenantCommonConfig(dto)
	if err != nil {
		for _, msg := range errMsgs {
			resp.Diagnostics.AddError("Tenant Config Validation Error", msg)
		}
		if len(errMsgs) == 0 {
			resp.Diagnostics.AddError("Tenant Config Validation Error", err.Error())
		}
		return
	}

	// If edm_enablement is REQUESTED, also call POST /v1/api/edm-dataset/enable
	if !plan.EdmEnablement.IsNull() && strings.EqualFold(plan.EdmEnablement.ValueString(), "REQUESTED") {
		var enableResp map[string]any
		if err := r.client.Post(ctx, "/v1/api/edm-dataset/enable", nil, map[string]any{}, &enableResp); err != nil {
			resp.Diagnostics.AddError("Error enabling EDM dataset", fmt.Sprintf("Failed to call POST /v1/api/edm-dataset/enable: %s", err))
			return
		}
	}

	var patchResp map[string]any
	if err := r.client.Patch(ctx, "/v1/api/tenant-common-config/update", nil, dto, &patchResp); err != nil {
		resp.Diagnostics.AddError("Error updating tenant configuration", fmt.Sprintf("Failed to call PATCH /v1/api/tenant-common-config/update: %s", err))
		return
	}

	var getResp dlpclient.TenantCommonConfigDTO
	if err := r.client.Get(ctx, "/v1/api/tenant-common-config/properties", nil, &getResp); err != nil {
		resp.Diagnostics.AddError("Error reading tenant configuration", fmt.Sprintf("Failed to call GET /v1/api/tenant-common-config/properties: %s", err))
		return
	}

	state := plan
	dlpTenantConfigMergeDTO(&state, &getResp)
	if state.Id.IsNull() || state.Id.ValueString() == "" {
		state.Id = types.StringValue("tenant_config")
	}
	state.Tfid = types.StringValue(state.Id.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dlpTenantConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.DlpTenantConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var getResp dlpclient.TenantCommonConfigDTO
	if err := r.client.Get(ctx, "/v1/api/tenant-common-config/properties", nil, &getResp); err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading tenant configuration", fmt.Sprintf("Failed to call GET /v1/api/tenant-common-config/properties: %s", err))
		return
	}

	dlpTenantConfigMergeDTO(&state, &getResp)
	if state.Id.IsNull() || state.Id.ValueString() == "" {
		state.Id = types.StringValue("tenant_config")
	}
	state.Tfid = types.StringValue(state.Id.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dlpTenantConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.DlpTenantConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state models.DlpTenantConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := dlpTenantConfigModelToDTO(&plan)

	errMsgs, err := dlpclient.ValidateTenantCommonConfig(dto)
	if err != nil {
		for _, msg := range errMsgs {
			resp.Diagnostics.AddError("Tenant Config Validation Error", msg)
		}
		if len(errMsgs) == 0 {
			resp.Diagnostics.AddError("Tenant Config Validation Error", err.Error())
		}
		return
	}

	if !plan.EdmEnablement.IsNull() && strings.EqualFold(plan.EdmEnablement.ValueString(), "REQUESTED") &&
		(state.EdmEnablement.IsNull() || !strings.EqualFold(state.EdmEnablement.ValueString(), "REQUESTED")) {
		var enableResp map[string]any
		if err := r.client.Post(ctx, "/v1/api/edm-dataset/enable", nil, map[string]any{}, &enableResp); err != nil {
			resp.Diagnostics.AddError("Error enabling EDM dataset", fmt.Sprintf("Failed to call POST /v1/api/edm-dataset/enable: %s", err))
			return
		}
	}

	var patchResp map[string]any
	if err := r.client.Patch(ctx, "/v1/api/tenant-common-config/update", nil, dto, &patchResp); err != nil {
		resp.Diagnostics.AddError("Error updating tenant configuration", fmt.Sprintf("Failed to call PATCH /v1/api/tenant-common-config/update: %s", err))
		return
	}

	var getResp dlpclient.TenantCommonConfigDTO
	if err := r.client.Get(ctx, "/v1/api/tenant-common-config/properties", nil, &getResp); err != nil {
		resp.Diagnostics.AddError("Error reading tenant configuration", fmt.Sprintf("Failed to call GET /v1/api/tenant-common-config/properties: %s", err))
		return
	}

	updatedState := plan
	dlpTenantConfigMergeDTO(&updatedState, &getResp)
	if updatedState.Id.IsNull() || updatedState.Id.ValueString() == "" {
		updatedState.Id = types.StringValue("tenant_config")
	}
	updatedState.Tfid = types.StringValue(updatedState.Id.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &updatedState)...)
}

func (r *dlpTenantConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting scm_dlp_tenant_config from Terraform state (singleton configuration)")
	resp.State.RemoveResource(ctx)
}

func (r *dlpTenantConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ============================================================================
// DATA SOURCE: scm_dlp_tenant_config
// ============================================================================

var (
	_ datasource.DataSource              = &dlpTenantConfigDataSource{}
	_ datasource.DataSourceWithConfigure = &dlpTenantConfigDataSource{}
)

func NewDlpTenantConfigDataSource() datasource.DataSource {
	return &dlpTenantConfigDataSource{}
}

type dlpTenantConfigDataSource struct {
	client *dlpclient.Client
}

func (d *dlpTenantConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_tenant_config"
}

func (d *dlpTenantConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DlpTenantConfigDataSourceSchema
}

func (d *dlpTenantConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *dlpTenantConfigDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getResp dlpclient.TenantCommonConfigDTO
	if err := d.client.Get(ctx, "/v1/api/tenant-common-config/properties", nil, &getResp); err != nil {
		resp.Diagnostics.AddError("Error reading tenant configuration", fmt.Sprintf("Failed to call GET /v1/api/tenant-common-config/properties: %s", err))
		return
	}

	var state models.DlpTenantConfigModel
	dlpTenantConfigMergeDTO(&state, &getResp)
	if state.Id.IsNull() || state.Id.ValueString() == "" {
		state.Id = types.StringValue("tenant_config")
	}
	state.Tfid = types.StringValue(state.Id.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ============================================================================
// Helpers
// ============================================================================

func dlpTenantConfigModelToDTO(m *models.DlpTenantConfigModel) *dlpclient.TenantCommonConfigDTO {
	dto := &dlpclient.TenantCommonConfigDTO{}
	if !m.Id.IsNull() && !m.Id.IsUnknown() {
		v := m.Id.ValueString()
		dto.ID = &v
	}
	if !m.EnableSnippets.IsNull() && !m.EnableSnippets.IsUnknown() {
		v := m.EnableSnippets.ValueBool()
		dto.EnableSnippets = &v
	}
	if !m.MaskLevel.IsNull() && !m.MaskLevel.IsUnknown() {
		v := m.MaskLevel.ValueString()
		dto.MaskLevel = &v
	}
	if !m.IncidentNotificationSnippetsEnablement.IsNull() && !m.IncidentNotificationSnippetsEnablement.IsUnknown() {
		v := m.IncidentNotificationSnippetsEnablement.ValueBool()
		dto.IncidentNotificationSnippetsEnablement = &v
	}
	if !m.EnableStructuredDataClustering.IsNull() && !m.EnableStructuredDataClustering.IsUnknown() {
		v := m.EnableStructuredDataClustering.ValueBool()
		dto.EnableStructuredDataClustering = &v
	}
	if !m.EnforceStructuredDataClusteringUniform.IsNull() && !m.EnforceStructuredDataClusteringUniform.IsUnknown() {
		v := m.EnforceStructuredDataClusteringUniform.ValueBool()
		dto.EnforceStructuredDataClusteringUniform = &v
	}
	if !m.StructuredDataClusteringMinUniformity.IsNull() && !m.StructuredDataClusteringMinUniformity.IsUnknown() {
		v := m.StructuredDataClusteringMinUniformity.ValueInt64()
		dto.StructuredDataClusteringMinUniformity = &v
	}
	if !m.EdmEnablement.IsNull() && !m.EdmEnablement.IsUnknown() {
		v := m.EdmEnablement.ValueString()
		dto.EdmEnablement = &v
	}
	if !m.EdmEnablementBanner.IsNull() && !m.EdmEnablementBanner.IsUnknown() {
		v := m.EdmEnablementBanner.ValueBool()
		dto.EdmEnablementBanner = &v
	}
	if !m.OcrEnablementBanner.IsNull() && !m.OcrEnablementBanner.IsUnknown() {
		v := m.OcrEnablementBanner.ValueBool()
		dto.OcrEnablementBanner = &v
	}
	// incident_notification_enablement, detection_exemption_period_minutes and
	// ocr_enablement are read-only on this resource; they are owned by
	// scm_dlp_alert_settings and scm_dlp_ocr_enablement respectively.
	return dto
}

func dlpTenantConfigMergeDTO(m *models.DlpTenantConfigModel, dto *dlpclient.TenantCommonConfigDTO) {
	if dto == nil {
		return
	}
	if dto.ID != nil {
		m.Id = types.StringValue(*dto.ID)
	}
	if dto.EnableSnippets != nil {
		m.EnableSnippets = types.BoolValue(*dto.EnableSnippets)
	} else if m.EnableSnippets.IsUnknown() {
		m.EnableSnippets = types.BoolNull()
	}
	if dto.MaskLevel != nil {
		m.MaskLevel = types.StringValue(*dto.MaskLevel)
	} else if m.MaskLevel.IsUnknown() {
		m.MaskLevel = types.StringNull()
	}
	if dto.IncidentNotificationEnablement != nil {
		m.IncidentNotificationEnablement = types.BoolValue(*dto.IncidentNotificationEnablement)
	} else if m.IncidentNotificationEnablement.IsUnknown() {
		m.IncidentNotificationEnablement = types.BoolNull()
	}
	if dto.IncidentNotificationSnippetsEnablement != nil {
		m.IncidentNotificationSnippetsEnablement = types.BoolValue(*dto.IncidentNotificationSnippetsEnablement)
	} else if m.IncidentNotificationSnippetsEnablement.IsUnknown() {
		m.IncidentNotificationSnippetsEnablement = types.BoolNull()
	}
	if dto.DetectionExemptionPeriodMinutes != nil {
		m.DetectionExemptionPeriodMinutes = types.Int64Value(*dto.DetectionExemptionPeriodMinutes)
	} else if m.DetectionExemptionPeriodMinutes.IsUnknown() {
		m.DetectionExemptionPeriodMinutes = types.Int64Null()
	}
	if dto.EnableStructuredDataClustering != nil {
		m.EnableStructuredDataClustering = types.BoolValue(*dto.EnableStructuredDataClustering)
	} else if m.EnableStructuredDataClustering.IsUnknown() {
		m.EnableStructuredDataClustering = types.BoolNull()
	}
	if dto.EnforceStructuredDataClusteringUniform != nil {
		m.EnforceStructuredDataClusteringUniform = types.BoolValue(*dto.EnforceStructuredDataClusteringUniform)
	} else if m.EnforceStructuredDataClusteringUniform.IsUnknown() {
		m.EnforceStructuredDataClusteringUniform = types.BoolNull()
	}
	if dto.StructuredDataClusteringMinUniformity != nil {
		m.StructuredDataClusteringMinUniformity = types.Int64Value(*dto.StructuredDataClusteringMinUniformity)
	} else if m.StructuredDataClusteringMinUniformity.IsUnknown() {
		m.StructuredDataClusteringMinUniformity = types.Int64Null()
	}
	if dto.EdmEnablement != nil {
		m.EdmEnablement = types.StringValue(*dto.EdmEnablement)
	} else if m.EdmEnablement.IsUnknown() {
		m.EdmEnablement = types.StringNull()
	}
	if dto.EdmEnablementBanner != nil {
		m.EdmEnablementBanner = types.BoolValue(*dto.EdmEnablementBanner)
	} else if m.EdmEnablementBanner.IsUnknown() {
		m.EdmEnablementBanner = types.BoolNull()
	}
	if dto.OcrEnablement != nil {
		m.OcrEnablement = types.StringValue(*dto.OcrEnablement)
	} else if m.OcrEnablement.IsUnknown() {
		m.OcrEnablement = types.StringNull()
	}
	if dto.OcrEnablementBanner != nil {
		m.OcrEnablementBanner = types.BoolValue(*dto.OcrEnablementBanner)
	} else if m.OcrEnablementBanner.IsUnknown() {
		m.OcrEnablementBanner = types.BoolNull()
	}
}
