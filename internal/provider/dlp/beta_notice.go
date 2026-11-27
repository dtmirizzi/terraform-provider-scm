package provider

// Enterprise DLP support is Beta / Experimental. Every scm_dlp_* resource,
// data source and action emits a warning from ValidateConfig, which Terraform
// calls only for blocks present in the configuration, so users who do not use
// DLP never see it. Terraform consolidates identical warnings into one.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

const (
	BetaWarningSummary = "Enterprise DLP support is Beta / Experimental"
	BetaWarningDetail  = "Enterprise DLP resources, data sources, and actions (scm_dlp_*) in terraform-provider-scm " +
		"are Beta / Experimental. Their schemas, attributes, and API behavior may change in future releases " +
		"without a major version bump. Pin the provider version and review release notes before upgrading."
)

func addBetaWarning(diags *diag.Diagnostics) {
	diags.AddWarning(BetaWarningSummary, BetaWarningDetail)
}

var (
	_ datasource.DataSourceWithValidateConfig = &DataPatternDataSource{}
	_ datasource.DataSourceWithValidateConfig = &DataPatternListDataSource{}
	_ resource.ResourceWithValidateConfig     = &DataPatternResource{}
	_ datasource.DataSourceWithValidateConfig = &DictionaryDataSource{}
	_ datasource.DataSourceWithValidateConfig = &DictionaryListDataSource{}
	_ resource.ResourceWithValidateConfig     = &DictionaryResource{}
	_ datasource.DataSourceWithValidateConfig = &DocumentTypeDataSource{}
	_ datasource.DataSourceWithValidateConfig = &DocumentTypeListDataSource{}
	_ resource.ResourceWithValidateConfig     = &DocumentTypeResource{}
	_ action.ActionWithValidateConfig         = &DryRunScanAction{}
	_ datasource.DataSourceWithValidateConfig = &DryRunScanDataSource{}
	_ datasource.DataSourceWithValidateConfig = &EdmDatasetDataSource{}
	_ datasource.DataSourceWithValidateConfig = &EdmDatasetListDataSource{}
	_ resource.ResourceWithValidateConfig     = &EdmDatasetResource{}
	_ datasource.DataSourceWithValidateConfig = &EndpointPolicyDataSource{}
	_ datasource.DataSourceWithValidateConfig = &EndpointPolicyListDataSource{}
	_ resource.ResourceWithValidateConfig     = &EndpointPolicyResource{}
	_ datasource.DataSourceWithValidateConfig = &ForwardProfileDataSource{}
	_ datasource.DataSourceWithValidateConfig = &ForwardProfileListDataSource{}
	_ resource.ResourceWithValidateConfig     = &ForwardProfileResource{}
	_ datasource.DataSourceWithValidateConfig = &PeripheralDataSource{}
	_ datasource.DataSourceWithValidateConfig = &PeripheralListDataSource{}
	_ resource.ResourceWithValidateConfig     = &PeripheralResource{}
	_ datasource.DataSourceWithValidateConfig = &SyslogProfileDataSource{}
	_ datasource.DataSourceWithValidateConfig = &SyslogProfileListDataSource{}
	_ resource.ResourceWithValidateConfig     = &SyslogProfileResource{}
	_ datasource.DataSourceWithValidateConfig = &dataFilteringProfileDataSource{}
	_ datasource.DataSourceWithValidateConfig = &dataFilteringProfileListDataSource{}
	_ resource.ResourceWithValidateConfig     = &dataFilteringProfileResource{}
	_ datasource.DataSourceWithValidateConfig = &dataProfileDataSource{}
	_ datasource.DataSourceWithValidateConfig = &dataProfileListDataSource{}
	_ resource.ResourceWithValidateConfig     = &dataProfileResource{}
	_ datasource.DataSourceWithValidateConfig = &dlpAlertSettingsDataSource{}
	_ resource.ResourceWithValidateConfig     = &dlpAlertSettingsResource{}
	_ datasource.DataSourceWithValidateConfig = &dlpOcrEnablementDataSource{}
	_ resource.ResourceWithValidateConfig     = &dlpOcrEnablementResource{}
	_ datasource.DataSourceWithValidateConfig = &dlpTenantChannelsDataSource{}
	_ datasource.DataSourceWithValidateConfig = &dlpTenantConfigDataSource{}
	_ resource.ResourceWithValidateConfig     = &dlpTenantConfigResource{}
)

func (r *DataPatternDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *DataPatternListDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *DataPatternResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *DictionaryDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *DictionaryListDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *DictionaryResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *DocumentTypeDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *DocumentTypeListDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *DocumentTypeResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *DryRunScanAction) ValidateConfig(_ context.Context, _ action.ValidateConfigRequest, resp *action.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *DryRunScanDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *EdmDatasetDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *EdmDatasetListDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *EdmDatasetResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *EndpointPolicyDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *EndpointPolicyListDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *EndpointPolicyResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *ForwardProfileDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *ForwardProfileListDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *ForwardProfileResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *PeripheralDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *PeripheralListDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *PeripheralResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *SyslogProfileDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *SyslogProfileListDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *SyslogProfileResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dataFilteringProfileDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dataFilteringProfileListDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dataFilteringProfileResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dataProfileDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dataProfileListDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dataProfileResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dlpAlertSettingsDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dlpAlertSettingsResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dlpOcrEnablementDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dlpOcrEnablementResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dlpTenantChannelsDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dlpTenantConfigDataSource) ValidateConfig(_ context.Context, _ datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}

func (r *dlpTenantConfigResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	addBetaWarning(&resp.Diagnostics)
}
