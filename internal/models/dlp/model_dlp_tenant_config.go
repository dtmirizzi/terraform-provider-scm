package models

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ============================================================================
// 1. scm_dlp_tenant_config
// ============================================================================

// DlpTenantConfigModel maps to /v1/api/tenant-common-config/{properties,update}.
type DlpTenantConfigModel struct {
	Id                                     types.String `tfsdk:"id"`
	Tfid                                   types.String `tfsdk:"tfid"`
	EnableSnippets                         types.Bool   `tfsdk:"enable_snippets"`
	MaskLevel                              types.String `tfsdk:"mask_level"`
	IncidentNotificationEnablement         types.Bool   `tfsdk:"incident_notification_enablement"`
	IncidentNotificationSnippetsEnablement types.Bool   `tfsdk:"incident_notification_snippets_enablement"`
	DetectionExemptionPeriodMinutes        types.Int64  `tfsdk:"detection_exemption_period_minutes"`
	EnableStructuredDataClustering         types.Bool   `tfsdk:"enable_structured_data_clustering"`
	EnforceStructuredDataClusteringUniform types.Bool   `tfsdk:"enforce_structured_data_clustering_uniformity"`
	StructuredDataClusteringMinUniformity  types.Int64  `tfsdk:"structured_data_clustering_minimum_uniformity"`
	EdmEnablement                          types.String `tfsdk:"edm_enablement"`
	EdmEnablementBanner                    types.Bool   `tfsdk:"edm_enablement_banner"`
	OcrEnablement                          types.String `tfsdk:"ocr_enablement"`
	OcrEnablementBanner                    types.Bool   `tfsdk:"ocr_enablement_banner"`
}

var DlpTenantConfigResourceSchema = rschema.Schema{
	Description: "Manages Enterprise DLP tenant-wide configuration including snippet capture, masking level, OCR, and alert exemption periods.",
	Attributes: map[string]rschema.Attribute{
		"id": rschema.StringAttribute{
			Description: "Tenant identifier.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"tfid": rschema.StringAttribute{
			Description: "Terraform identifier.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"enable_snippets": rschema.BoolAttribute{
			Description: "Enable or disable snippet capture during DLP scans.",
			Optional:    true,
			Computed:    true,
		},
		"mask_level": rschema.StringAttribute{
			Description: "Snippet masking level. Valid values are NO_MASK, PARTIAL_MASK, FULL_MASK.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("NO_MASK", "PARTIAL_MASK", "FULL_MASK"),
			},
		},
		"incident_notification_enablement": rschema.BoolAttribute{
			Description: "Enable incident notifications for the tenant.",
			Optional:    true,
			Computed:    true,
		},
		"incident_notification_snippets_enablement": rschema.BoolAttribute{
			Description: "Include snippets in incident notifications.",
			Optional:    true,
			Computed:    true,
		},
		"detection_exemption_period_minutes": rschema.Int64Attribute{
			Description: "Alert detection exemption period in minutes (1 to 129600).",
			Optional:    true,
			Computed:    true,
			Validators: []validator.Int64{
				int64validator.Between(1, 129600),
			},
		},
		"enable_structured_data_clustering": rschema.BoolAttribute{
			Description: "Enable structured data clustering for the tenant.",
			Optional:    true,
			Computed:    true,
		},
		"enforce_structured_data_clustering_uniformity": rschema.BoolAttribute{
			Description: "Enforce uniformity on structured data clustering.",
			Optional:    true,
			Computed:    true,
		},
		"structured_data_clustering_minimum_uniformity": rschema.Int64Attribute{
			Description: "Minimum uniformity score for structured data clustering.",
			Optional:    true,
			Computed:    true,
		},
		"edm_enablement": rschema.StringAttribute{
			Description: "Exact Data Match (EDM) enablement state. Valid values: REQUESTED, ENABLED, DISABLED.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("REQUESTED", "ENABLED", "DISABLED"),
			},
		},
		"edm_enablement_banner": rschema.BoolAttribute{
			Description: "Whether EDM enablement banner is displayed.",
			Optional:    true,
			Computed:    true,
		},
		"ocr_enablement": rschema.StringAttribute{
			Description: "OCR enablement state for the tenant. Valid values: REQUESTED, ENABLED, DISABLED.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("REQUESTED", "ENABLED", "DISABLED"),
			},
		},
		"ocr_enablement_banner": rschema.BoolAttribute{
			Description: "Whether OCR enablement banner is displayed.",
			Optional:    true,
			Computed:    true,
		},
	},
}

var DlpTenantConfigDataSourceSchema = dsschema.Schema{
	Description: "Retrieves Enterprise DLP tenant-wide configuration.",
	Attributes: map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			Description: "Tenant identifier.",
			Computed:    true,
		},
		"tfid": dsschema.StringAttribute{
			Description: "Terraform identifier.",
			Computed:    true,
		},
		"enable_snippets": dsschema.BoolAttribute{
			Description: "Enable or disable snippet capture during DLP scans.",
			Computed:    true,
		},
		"mask_level": dsschema.StringAttribute{
			Description: "Snippet masking level (NO_MASK, PARTIAL_MASK, FULL_MASK).",
			Computed:    true,
		},
		"incident_notification_enablement": dsschema.BoolAttribute{
			Description: "Enable incident notifications for the tenant.",
			Computed:    true,
		},
		"incident_notification_snippets_enablement": dsschema.BoolAttribute{
			Description: "Include snippets in incident notifications.",
			Computed:    true,
		},
		"detection_exemption_period_minutes": dsschema.Int64Attribute{
			Description: "Alert detection exemption period in minutes.",
			Computed:    true,
		},
		"enable_structured_data_clustering": dsschema.BoolAttribute{
			Description: "Enable structured data clustering for the tenant.",
			Computed:    true,
		},
		"enforce_structured_data_clustering_uniformity": dsschema.BoolAttribute{
			Description: "Enforce uniformity on structured data clustering.",
			Computed:    true,
		},
		"structured_data_clustering_minimum_uniformity": dsschema.Int64Attribute{
			Description: "Minimum uniformity score for structured data clustering.",
			Computed:    true,
		},
		"edm_enablement": dsschema.StringAttribute{
			Description: "Exact Data Match (EDM) enablement state.",
			Computed:    true,
		},
		"edm_enablement_banner": dsschema.BoolAttribute{
			Description: "Whether EDM enablement banner is displayed.",
			Computed:    true,
		},
		"ocr_enablement": dsschema.StringAttribute{
			Description: "OCR enablement state for the tenant.",
			Computed:    true,
		},
		"ocr_enablement_banner": dsschema.BoolAttribute{
			Description: "Whether OCR enablement banner is displayed.",
			Computed:    true,
		},
	},
}

// ============================================================================
// 2. scm_dlp_ocr_enablement
// ============================================================================

// DlpOcrEnablementModel maps to /v2/api/ocr.
type DlpOcrEnablementModel struct {
	TargetServiceName types.String `tfsdk:"target_service_name"`
	OcrEnablement     types.String `tfsdk:"ocr_enablement"`
	TenantId          types.String `tfsdk:"tenant_id"`
	Tfid              types.String `tfsdk:"tfid"`
}

var DlpOcrEnablementResourceSchema = rschema.Schema{
	Description: "Manages Enterprise DLP Optical Character Recognition (OCR) enablement per target service.",
	Attributes: map[string]rschema.Attribute{
		"target_service_name": rschema.StringAttribute{
			Description: "Target service name (e.g. prisma-access, ngfw, dlp).",
			Required:    true,
			Validators: []validator.String{
				stringvalidator.OneOf(
					"gpcs",
					"aperture",
					"redlock",
					"prisma-cloud",
					"prisma-saas",
					"prisma-access",
					"ngfw",
					"dlp",
					"email-dlp",
					"cortex-xdr",
					"endpoint-dlp",
					"prisma-access-browser",
				),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"ocr_enablement": rschema.StringAttribute{
			Description: "OCR enablement state. Valid values: REQUESTED, ENABLED, DISABLED.",
			Required:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("REQUESTED", "ENABLED", "DISABLED"),
			},
		},
		"tenant_id": rschema.StringAttribute{
			Description: "Tenant identifier.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"tfid": rschema.StringAttribute{
			Description: "Terraform identifier.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	},
}

var DlpOcrEnablementDataSourceSchema = dsschema.Schema{
	Description: "Retrieves Enterprise DLP OCR enablement state for a target service.",
	Attributes: map[string]dsschema.Attribute{
		"target_service_name": dsschema.StringAttribute{
			Description: "Target service name (e.g. prisma-access, ngfw, dlp).",
			Required:    true,
		},
		"ocr_enablement": dsschema.StringAttribute{
			Description: "OCR enablement state (REQUESTED, ENABLED, DISABLED).",
			Computed:    true,
		},
		"tenant_id": dsschema.StringAttribute{
			Description: "Tenant identifier.",
			Computed:    true,
		},
		"tfid": dsschema.StringAttribute{
			Description: "Terraform identifier.",
			Computed:    true,
		},
	},
}

// ============================================================================
// 3. scm_dlp_alert_settings
// ============================================================================

// DlpAlertSettingsModel maps to /v1/api/alerts/setting.
type DlpAlertSettingsModel struct {
	Id                              types.String `tfsdk:"id"`
	Tfid                            types.String `tfsdk:"tfid"`
	IncidentNotificationEnablement  types.Bool   `tfsdk:"incident_notification_enablement"`
	DetectionExemptionPeriodMinutes types.Int64  `tfsdk:"detection_exemption_period_minutes"`
}

var DlpAlertSettingsResourceSchema = rschema.Schema{
	Description: "Manages Enterprise DLP alert settings.",
	Attributes: map[string]rschema.Attribute{
		"id": rschema.StringAttribute{
			Description: "Alert settings identifier.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"tfid": rschema.StringAttribute{
			Description: "Terraform identifier.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"incident_notification_enablement": rschema.BoolAttribute{
			Description: "Enable incident notifications.",
			Optional:    true,
			Computed:    true,
		},
		"detection_exemption_period_minutes": rschema.Int64Attribute{
			Description: "Alert detection exemption period in minutes (1 to 129600).",
			Optional:    true,
			Computed:    true,
			Validators: []validator.Int64{
				int64validator.Between(1, 129600),
			},
		},
	},
}

var DlpAlertSettingsDataSourceSchema = dsschema.Schema{
	Description: "Retrieves Enterprise DLP alert settings.",
	Attributes: map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			Description: "Alert settings identifier.",
			Computed:    true,
		},
		"tfid": dsschema.StringAttribute{
			Description: "Terraform identifier.",
			Computed:    true,
		},
		"incident_notification_enablement": dsschema.BoolAttribute{
			Description: "Enable incident notifications.",
			Computed:    true,
		},
		"detection_exemption_period_minutes": dsschema.Int64Attribute{
			Description: "Alert detection exemption period in minutes.",
			Computed:    true,
		},
	},
}

// ============================================================================
// 4. scm_dlp_tenant_channels
// ============================================================================

// DlpTenantChannelsModel maps to /v1/api/channels.
type DlpTenantChannelsModel struct {
	Tfid     types.String `tfsdk:"tfid"`
	Channels types.Map    `tfsdk:"channels"`
}

var DlpTenantChannelsDataSourceSchema = dsschema.Schema{
	Description: "Retrieves effective/licensed Enterprise DLP channels for the tenant.",
	Attributes: map[string]dsschema.Attribute{
		"tfid": dsschema.StringAttribute{
			Description: "Terraform identifier.",
			Computed:    true,
		},
		"channels": dsschema.MapAttribute{
			Description: "Map of channel name to enablement flag (e.g. prisma-access: true).",
			ElementType: types.BoolType,
			Computed:    true,
		},
	},
}
