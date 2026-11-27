package models

import (
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ============================================================================
// 1. Endpoint Policy (scm_dlp_endpoint_policy)
// ============================================================================

// EndpointPolicyModel represents the Terraform model for scm_dlp_endpoint_policy.
type EndpointPolicyModel struct {
	Tfid         types.String         `tfsdk:"tfid"`
	Id           types.String         `tfsdk:"id"`
	Name         types.String         `tfsdk:"name"`
	Description  types.String         `tfsdk:"description"`
	Enabled      types.Bool           `tfsdk:"enabled"`
	Type         types.String         `tfsdk:"type"`
	Severity     types.String         `tfsdk:"severity"`
	Priority     types.Int64          `tfsdk:"priority"`
	ScopeJson    jsontypes.Normalized `tfsdk:"scope_json"`
	ResponseJson jsontypes.Normalized `tfsdk:"response_json"`
}

// EndpointPolicyListModel represents the data source model for scm_dlp_endpoint_policy_list.
type EndpointPolicyListModel struct {
	Total types.Int64           `tfsdk:"total"`
	Data  []EndpointPolicyModel `tfsdk:"data"`
}

var EndpointPolicyResourceSchema = rschema.Schema{
	Description: "[Beta / Experimental] Manages an Enterprise DLP Endpoint Policy.",
	Attributes: map[string]rschema.Attribute{
		"tfid": rschema.StringAttribute{
			Computed:    true,
			Description: "The Terraform ID.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": rschema.StringAttribute{
			Computed:    true,
			Description: "The unique identifier of the endpoint policy.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": rschema.StringAttribute{
			Required:    true,
			Description: "The name of the endpoint policy.",
		},
		"description": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Description of the endpoint policy.",
		},
		"enabled": rschema.BoolAttribute{
			Required:    true,
			Description: "Whether the endpoint policy is enabled.",
		},
		"type": rschema.StringAttribute{
			Required:    true,
			Description: "Policy type: endpoint_peripheral_control, endpoint_data_in_motion, or network.",
		},
		"severity": rschema.StringAttribute{
			Required:    true,
			Description: "Severity level: informational, low, medium, high, or critical.",
		},
		"priority": rschema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: "Priority order of the endpoint policy. Assigned by the backend when omitted.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"scope_json": rschema.StringAttribute{
			CustomType:  jsontypes.NormalizedType{},
			Required:    true,
			Description: "JSON object representing the policy scope.",
		},
		"response_json": rschema.StringAttribute{
			CustomType:  jsontypes.NormalizedType{},
			Required:    true,
			Description: "JSON object representing the policy response/action.",
		},
	},
}

var EndpointPolicyDataSourceSchema = dsschema.Schema{
	Description: "[Beta / Experimental] Retrieves details of an Enterprise DLP Endpoint Policy.",
	Attributes: map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			Required:    true,
			Description: "The unique identifier of the endpoint policy.",
		},
		"tfid": dsschema.StringAttribute{
			Computed:    true,
			Description: "The Terraform ID.",
		},
		"name": dsschema.StringAttribute{
			Computed:    true,
			Description: "The name of the endpoint policy.",
		},
		"description": dsschema.StringAttribute{
			Computed:    true,
			Description: "Description of the endpoint policy.",
		},
		"enabled": dsschema.BoolAttribute{
			Computed:    true,
			Description: "Whether the endpoint policy is enabled.",
		},
		"type": dsschema.StringAttribute{
			Computed:    true,
			Description: "Policy type: endpoint_peripheral_control, endpoint_data_in_motion, or network.",
		},
		"severity": dsschema.StringAttribute{
			Computed:    true,
			Description: "Severity level: informational, low, medium, high, or critical.",
		},
		"priority": dsschema.Int64Attribute{
			Computed:    true,
			Description: "Priority order of the endpoint policy.",
		},
		"scope_json": dsschema.StringAttribute{
			CustomType:  jsontypes.NormalizedType{},
			Computed:    true,
			Description: "JSON object representing the policy scope.",
		},
		"response_json": dsschema.StringAttribute{
			CustomType:  jsontypes.NormalizedType{},
			Computed:    true,
			Description: "JSON object representing the policy response/action.",
		},
	},
}

var EndpointPolicyListDataSourceSchema = dsschema.Schema{
	Description: "[Beta / Experimental] Retrieves a list of Enterprise DLP Endpoint Policies.",
	Attributes: map[string]dsschema.Attribute{
		"total": dsschema.Int64Attribute{
			Computed:    true,
			Description: "Total number of endpoint policies found.",
		},
		"data": dsschema.ListNestedAttribute{
			Computed:    true,
			Description: "List of endpoint policies.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"tfid": dsschema.StringAttribute{
						Computed:    true,
						Description: "The Terraform ID.",
					},
					"id": dsschema.StringAttribute{
						Computed:    true,
						Description: "The unique identifier of the endpoint policy.",
					},
					"name": dsschema.StringAttribute{
						Computed:    true,
						Description: "The name of the endpoint policy.",
					},
					"description": dsschema.StringAttribute{
						Computed:    true,
						Description: "Description of the endpoint policy.",
					},
					"enabled": dsschema.BoolAttribute{
						Computed:    true,
						Description: "Whether the endpoint policy is enabled.",
					},
					"type": dsschema.StringAttribute{
						Computed:    true,
						Description: "Policy type: endpoint_peripheral_control, endpoint_data_in_motion, or network.",
					},
					"severity": dsschema.StringAttribute{
						Computed:    true,
						Description: "Severity level: informational, low, medium, high, or critical.",
					},
					"priority": dsschema.Int64Attribute{
						Computed:    true,
						Description: "Priority order of the endpoint policy.",
					},
					"scope_json": dsschema.StringAttribute{
						CustomType:  jsontypes.NormalizedType{},
						Computed:    true,
						Description: "JSON object representing the policy scope.",
					},
					"response_json": dsschema.StringAttribute{
						CustomType:  jsontypes.NormalizedType{},
						Computed:    true,
						Description: "JSON object representing the policy response/action.",
					},
				},
			},
		},
	},
}

// ============================================================================
// 2. Peripheral (scm_dlp_peripheral)
// ============================================================================

// PeripheralModel represents the Terraform model for scm_dlp_peripheral.
type PeripheralModel struct {
	Tfid          types.String `tfsdk:"tfid"`
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Type          types.String `tfsdk:"type"`
	PrinterType   types.String `tfsdk:"printer_type"`
	ServerAddress types.String `tfsdk:"server_address"`
	VendorId      types.String `tfsdk:"vendor_id"`
	ProductId     types.String `tfsdk:"product_id"`
}

// PeripheralListModel represents the data source model for scm_dlp_peripheral_list.
type PeripheralListModel struct {
	Total types.Int64       `tfsdk:"total"`
	Data  []PeripheralModel `tfsdk:"data"`
}

var PeripheralResourceSchema = rschema.Schema{
	Description: "[Beta / Experimental] Manages an Enterprise DLP Peripheral.",
	Attributes: map[string]rschema.Attribute{
		"tfid": rschema.StringAttribute{
			Computed:    true,
			Description: "The Terraform ID.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": rschema.StringAttribute{
			Computed:    true,
			Description: "The unique identifier of the peripheral.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": rschema.StringAttribute{
			Required:    true,
			Description: "The name of the peripheral.",
		},
		"description": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Description of the peripheral.",
		},
		"type": rschema.StringAttribute{
			Required:    true,
			Description: "Peripheral type: usb, printer, or network_share.",
		},
		"printer_type": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Printer type (for printer peripherals).",
		},
		"server_address": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Server address (for network share peripherals).",
		},
		"vendor_id": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Vendor ID (for USB peripherals).",
		},
		"product_id": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Product ID (for USB peripherals).",
		},
	},
}

var PeripheralDataSourceSchema = dsschema.Schema{
	Description: "[Beta / Experimental] Retrieves details of an Enterprise DLP Peripheral.",
	Attributes: map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			Required:    true,
			Description: "The unique identifier of the peripheral.",
		},
		"tfid": dsschema.StringAttribute{
			Computed:    true,
			Description: "The Terraform ID.",
		},
		"name": dsschema.StringAttribute{
			Computed:    true,
			Description: "The name of the peripheral.",
		},
		"description": dsschema.StringAttribute{
			Computed:    true,
			Description: "Description of the peripheral.",
		},
		"type": dsschema.StringAttribute{
			Computed:    true,
			Description: "Peripheral type: usb, printer, or network_share.",
		},
		"printer_type": dsschema.StringAttribute{
			Computed:    true,
			Description: "Printer type (for printer peripherals).",
		},
		"server_address": dsschema.StringAttribute{
			Computed:    true,
			Description: "Server address (for network share peripherals).",
		},
		"vendor_id": dsschema.StringAttribute{
			Computed:    true,
			Description: "Vendor ID (for USB peripherals).",
		},
		"product_id": dsschema.StringAttribute{
			Computed:    true,
			Description: "Product ID (for USB peripherals).",
		},
	},
}

var PeripheralListDataSourceSchema = dsschema.Schema{
	Description: "[Beta / Experimental] Retrieves a list of Enterprise DLP Peripherals.",
	Attributes: map[string]dsschema.Attribute{
		"total": dsschema.Int64Attribute{
			Computed:    true,
			Description: "Total number of peripherals found.",
		},
		"data": dsschema.ListNestedAttribute{
			Computed:    true,
			Description: "List of peripherals.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"tfid": dsschema.StringAttribute{
						Computed:    true,
						Description: "The Terraform ID.",
					},
					"id": dsschema.StringAttribute{
						Computed:    true,
						Description: "The unique identifier of the peripheral.",
					},
					"name": dsschema.StringAttribute{
						Computed:    true,
						Description: "The name of the peripheral.",
					},
					"description": dsschema.StringAttribute{
						Computed:    true,
						Description: "Description of the peripheral.",
					},
					"type": dsschema.StringAttribute{
						Computed:    true,
						Description: "Peripheral type: usb, printer, or network_share.",
					},
					"printer_type": dsschema.StringAttribute{
						Computed:    true,
						Description: "Printer type (for printer peripherals).",
					},
					"server_address": dsschema.StringAttribute{
						Computed:    true,
						Description: "Server address (for network share peripherals).",
					},
					"vendor_id": dsschema.StringAttribute{
						Computed:    true,
						Description: "Vendor ID (for USB peripherals).",
					},
					"product_id": dsschema.StringAttribute{
						Computed:    true,
						Description: "Product ID (for USB peripherals).",
					},
				},
			},
		},
	},
}

// ============================================================================
// 3. Syslog Profile (scm_dlp_syslog_profile)
// ============================================================================

// SyslogProfileModel represents the Terraform model for scm_dlp_syslog_profile.
type SyslogProfileModel struct {
	Tfid           types.String `tfsdk:"tfid"`
	Id             types.String `tfsdk:"id"`
	ProfileName    types.String `tfsdk:"profile_name"`
	Address        types.String `tfsdk:"address"`
	Port           types.Int64  `tfsdk:"port"`
	AlertEmail     types.String `tfsdk:"alert_email"`
	Format         types.String `tfsdk:"format"`
	ConnectionType types.String `tfsdk:"connection_type"`
	Cert           types.String `tfsdk:"cert"`
}

// SyslogProfileListModel represents the data source model for scm_dlp_syslog_profile_list.
type SyslogProfileListModel struct {
	Total types.Int64          `tfsdk:"total"`
	Data  []SyslogProfileModel `tfsdk:"data"`
}

var SyslogProfileResourceSchema = rschema.Schema{
	Description: "[Beta / Experimental] Manages an Enterprise DLP Syslog Server Profile.",
	Attributes: map[string]rschema.Attribute{
		"tfid": rschema.StringAttribute{
			Computed:    true,
			Description: "The Terraform ID.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": rschema.StringAttribute{
			Computed:    true,
			Description: "The unique identifier of the syslog server profile.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"profile_name": rschema.StringAttribute{
			Required:    true,
			Description: "The name of the syslog profile.",
		},
		"address": rschema.StringAttribute{
			Required:    true,
			Description: "IP address or hostname of the syslog server.",
		},
		"port": rschema.Int64Attribute{
			Required:    true,
			Description: "Port number of the syslog server.",
		},
		"alert_email": rschema.StringAttribute{
			Required:    true,
			Description: "Email address to receive alerts for syslog delivery failures.",
		},
		"format": rschema.StringAttribute{
			Required:    true,
			Description: "Syslog format: IETF, CSV, LEEF, CEF, or LEEF 1.0.",
		},
		"connection_type": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Connection type: TCP or UDP.",
		},
		"cert": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Sensitive:   true,
			Description: "Certificate content for secure (TLS) syslog connections. Marked sensitive so it is not shown in plan output.",
		},
	},
}

var SyslogProfileDataSourceSchema = dsschema.Schema{
	Description: "[Beta / Experimental] Retrieves details of an Enterprise DLP Syslog Server Profile.",
	Attributes: map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			Required:    true,
			Description: "The unique identifier of the syslog server profile.",
		},
		"tfid": dsschema.StringAttribute{
			Computed:    true,
			Description: "The Terraform ID.",
		},
		"profile_name": dsschema.StringAttribute{
			Computed:    true,
			Description: "The name of the syslog profile.",
		},
		"address": dsschema.StringAttribute{
			Computed:    true,
			Description: "IP address or hostname of the syslog server.",
		},
		"port": dsschema.Int64Attribute{
			Computed:    true,
			Description: "Port number of the syslog server.",
		},
		"alert_email": dsschema.StringAttribute{
			Computed:    true,
			Description: "Email address to receive alerts for syslog delivery failures.",
		},
		"format": dsschema.StringAttribute{
			Computed:    true,
			Description: "Syslog format: IETF, CSV, LEEF, CEF, or LEEF 1.0.",
		},
		"connection_type": dsschema.StringAttribute{
			Computed:    true,
			Description: "Connection type: TCP or UDP.",
		},
		"cert": dsschema.StringAttribute{
			Computed:    true,
			Sensitive:   true,
			Description: "Certificate content for secure connection.",
		},
	},
}

var SyslogProfileListDataSourceSchema = dsschema.Schema{
	Description: "[Beta / Experimental] Retrieves a list of Enterprise DLP Syslog Server Profiles.",
	Attributes: map[string]dsschema.Attribute{
		"total": dsschema.Int64Attribute{
			Computed:    true,
			Description: "Total number of syslog server profiles found.",
		},
		"data": dsschema.ListNestedAttribute{
			Computed:    true,
			Description: "List of syslog server profiles.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"tfid": dsschema.StringAttribute{
						Computed:    true,
						Description: "The Terraform ID.",
					},
					"id": dsschema.StringAttribute{
						Computed:    true,
						Description: "The unique identifier of the syslog server profile.",
					},
					"profile_name": dsschema.StringAttribute{
						Computed:    true,
						Description: "The name of the syslog profile.",
					},
					"address": dsschema.StringAttribute{
						Computed:    true,
						Description: "IP address or hostname of the syslog server.",
					},
					"port": dsschema.Int64Attribute{
						Computed:    true,
						Description: "Port number of the syslog server.",
					},
					"alert_email": dsschema.StringAttribute{
						Computed:    true,
						Description: "Email address to receive alerts for syslog delivery failures.",
					},
					"format": dsschema.StringAttribute{
						Computed:    true,
						Description: "Syslog format: IETF, CSV, LEEF, CEF, or LEEF 1.0.",
					},
					"connection_type": dsschema.StringAttribute{
						Computed:    true,
						Description: "Connection type: TCP or UDP.",
					},
					"cert": dsschema.StringAttribute{
						Computed:    true,
						Sensitive:   true,
						Description: "Certificate content for secure connection.",
					},
				},
			},
		},
	},
}

// ============================================================================
// 4. Forward Profile (scm_dlp_forward_profile)
// ============================================================================

// ForwardProfileModel represents the Terraform model for scm_dlp_forward_profile.
type ForwardProfileModel struct {
	Tfid    types.String `tfsdk:"tfid"`
	Id      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
	Regions types.List   `tfsdk:"regions"` // types.List of types.String
}

// ForwardProfileListModel represents the data source model for scm_dlp_forward_profile_list.
type ForwardProfileListModel struct {
	Total types.Int64           `tfsdk:"total"`
	Data  []ForwardProfileModel `tfsdk:"data"`
}

var ForwardProfileResourceSchema = rschema.Schema{
	Description: "[Beta / Experimental] Manages an Enterprise DLP Forward Profile.",
	Attributes: map[string]rschema.Attribute{
		"tfid": rschema.StringAttribute{
			Computed:    true,
			Description: "The Terraform ID.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": rschema.StringAttribute{
			Computed:    true,
			Description: "The unique identifier of the forward profile.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": rschema.StringAttribute{
			Required:    true,
			Description: "The name of the forward profile.",
		},
		"enabled": rschema.BoolAttribute{
			Required:    true,
			Description: "Whether the forward profile is enabled.",
		},
		"regions": rschema.ListAttribute{
			ElementType: types.StringType,
			Optional:    true,
			Description: "List of regions to filter forwarded logs (e.g. us, eu, ap, etc.).",
		},
	},
}

var ForwardProfileDataSourceSchema = dsschema.Schema{
	Description: "[Beta / Experimental] Retrieves details of an Enterprise DLP Forward Profile.",
	Attributes: map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			Required:    true,
			Description: "The unique identifier of the forward profile.",
		},
		"tfid": dsschema.StringAttribute{
			Computed:    true,
			Description: "The Terraform ID.",
		},
		"name": dsschema.StringAttribute{
			Computed:    true,
			Description: "The name of the forward profile.",
		},
		"enabled": dsschema.BoolAttribute{
			Computed:    true,
			Description: "Whether the forward profile is enabled.",
		},
		"regions": dsschema.ListAttribute{
			ElementType: types.StringType,
			Computed:    true,
			Description: "List of regions to filter forwarded logs.",
		},
	},
}

var ForwardProfileListDataSourceSchema = dsschema.Schema{
	Description: "[Beta / Experimental] Retrieves a list of Enterprise DLP Forward Profiles.",
	Attributes: map[string]dsschema.Attribute{
		"total": dsschema.Int64Attribute{
			Computed:    true,
			Description: "Total number of forward profiles found.",
		},
		"data": dsschema.ListNestedAttribute{
			Computed:    true,
			Description: "List of forward profiles.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"tfid": dsschema.StringAttribute{
						Computed:    true,
						Description: "The Terraform ID.",
					},
					"id": dsschema.StringAttribute{
						Computed:    true,
						Description: "The unique identifier of the forward profile.",
					},
					"name": dsschema.StringAttribute{
						Computed:    true,
						Description: "The name of the forward profile.",
					},
					"enabled": dsschema.BoolAttribute{
						Computed:    true,
						Description: "Whether the forward profile is enabled.",
					},
					"regions": dsschema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
						Description: "List of regions to filter forwarded logs.",
					},
				},
			},
		},
	},
}
