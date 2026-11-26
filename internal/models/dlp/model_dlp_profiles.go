package models

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ---------------------------------------------------------------------------
// 1. Data Profile Models & Schemas
// ---------------------------------------------------------------------------

// DataProfileRuleItemModel maps rule_item inside an expression tree.
type DataProfileRuleItemModel struct {
	Id                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	DetectionTechnique     types.String `tfsdk:"detection_technique"`
	OccurrenceOperatorType types.String `tfsdk:"occurrence_operator_type"`
	OccurrenceCount        types.Int64  `tfsdk:"occurrence_count"`
	OccurrenceLow          types.Int64  `tfsdk:"occurrence_low"`
	OccurrenceHigh         types.Int64  `tfsdk:"occurrence_high"`
	ConfidenceLevel        types.String `tfsdk:"confidence_level"`
	MatchType              types.String `tfsdk:"match_type"`
	ByUniqueCount          types.Bool   `tfsdk:"by_unique_count"`
}

func (m DataProfileRuleItemModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                       basetypes.StringType{},
		"name":                     basetypes.StringType{},
		"detection_technique":      basetypes.StringType{},
		"occurrence_operator_type": basetypes.StringType{},
		"occurrence_count":         basetypes.Int64Type{},
		"occurrence_low":           basetypes.Int64Type{},
		"occurrence_high":          basetypes.Int64Type{},
		"confidence_level":         basetypes.StringType{},
		"match_type":               basetypes.StringType{},
		"by_unique_count":          basetypes.BoolType{},
	}
}

// DataProfileSubExpressionLevel2Model represents sub_expressions at level 2 (leaf rule_item or operator with leaf).
type DataProfileSubExpressionLevel2Model struct {
	OperatorType types.String              `tfsdk:"operator_type"`
	RuleItem     *DataProfileRuleItemModel `tfsdk:"rule_item"`
}

func (m DataProfileSubExpressionLevel2Model) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"operator_type": basetypes.StringType{},
		"rule_item":     basetypes.ObjectType{AttrTypes: DataProfileRuleItemModel{}.AttrTypes()},
	}
}

// DataProfileSubExpressionModel represents sub_expressions at level 1.
type DataProfileSubExpressionModel struct {
	OperatorType   types.String                          `tfsdk:"operator_type"`
	RuleItem       *DataProfileRuleItemModel             `tfsdk:"rule_item"`
	SubExpressions []DataProfileSubExpressionLevel2Model `tfsdk:"sub_expressions"`
}

func (m DataProfileSubExpressionModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"operator_type":   basetypes.StringType{},
		"rule_item":       basetypes.ObjectType{AttrTypes: DataProfileRuleItemModel{}.AttrTypes()},
		"sub_expressions": basetypes.ListType{ElemType: basetypes.ObjectType{AttrTypes: DataProfileSubExpressionLevel2Model{}.AttrTypes()}},
	}
}

// DataProfileExpressionTreeModel represents the top-level expression_tree.
type DataProfileExpressionTreeModel struct {
	OperatorType       types.String                    `tfsdk:"operator_type"`
	RuleItem           *DataProfileRuleItemModel       `tfsdk:"rule_item"`
	SubExpressions     []DataProfileSubExpressionModel `tfsdk:"sub_expressions"`
	ExpressionTreeJson types.String                    `tfsdk:"expression_tree_json"`
}

func (m DataProfileExpressionTreeModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"operator_type":        basetypes.StringType{},
		"rule_item":            basetypes.ObjectType{AttrTypes: DataProfileRuleItemModel{}.AttrTypes()},
		"sub_expressions":      basetypes.ListType{ElemType: basetypes.ObjectType{AttrTypes: DataProfileSubExpressionModel{}.AttrTypes()}},
		"expression_tree_json": basetypes.StringType{},
	}
}

// DataProfileMultiProfileModel represents multi_profile rules.
type DataProfileMultiProfileModel struct {
	OperatorType   types.String  `tfsdk:"operator_type"`
	DataProfileIds []types.Int64 `tfsdk:"data_profile_ids"`
}

func (m DataProfileMultiProfileModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"operator_type":    basetypes.StringType{},
		"data_profile_ids": basetypes.ListType{ElemType: basetypes.Int64Type{}},
	}
}

// DataProfileDetectionRuleModel represents a single detection rule.
type DataProfileDetectionRuleModel struct {
	RuleType       types.String                    `tfsdk:"rule_type"`
	MultiProfile   *DataProfileMultiProfileModel   `tfsdk:"multi_profile"`
	ExpressionTree *DataProfileExpressionTreeModel `tfsdk:"expression_tree"`
}

func (m DataProfileDetectionRuleModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"rule_type":       basetypes.StringType{},
		"multi_profile":   basetypes.ObjectType{AttrTypes: DataProfileMultiProfileModel{}.AttrTypes()},
		"expression_tree": basetypes.ObjectType{AttrTypes: DataProfileExpressionTreeModel{}.AttrTypes()},
	}
}

// DataProfileResourceModel is the top-level resource model for scm_dlp_data_profile.
type DataProfileResourceModel struct {
	Tfid                  types.String                    `tfsdk:"tfid"`
	Id                    types.String                    `tfsdk:"id"`
	Name                  types.String                    `tfsdk:"name"`
	Description           types.String                    `tfsdk:"description"`
	ProfileType           types.String                    `tfsdk:"profile_type"`
	ProfileStatus         types.String                    `tfsdk:"profile_status"`
	IsGranularDataProfile types.Bool                      `tfsdk:"is_granular_data_profile"`
	Version               types.Int64                     `tfsdk:"version"`
	DetectionRules        []DataProfileDetectionRuleModel `tfsdk:"detection_rules"`
}

// RuleItemResourceAttributes returns schema attributes for rule_item.
func RuleItemResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The ID of the detection primitive (data pattern, dictionary, document type, ML model, etc.).",
			Required:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Descriptive name of the rule item.",
			Optional:            true,
			Computed:            true,
		},
		"detection_technique": schema.StringAttribute{
			MarkdownDescription: "The detection technique used (e.g. `regex`, `weighted_regex`, `ml`, `edm`, `dictionary`, `document_fingerprint`, `trainable_classifier`, `ml_document`, `file_property`, `pab`).",
			Required:            true,
		},
		"occurrence_operator_type": schema.StringAttribute{
			MarkdownDescription: "Occurrence operator: `any`, `less_than_equal_to`, `more_than_equal_to`, `between`.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("any", "less_than_equal_to", "more_than_equal_to", "between"),
			},
		},
		"occurrence_count": schema.Int64Attribute{
			MarkdownDescription: "Match count threshold when operator is `more_than_equal_to` or `less_than_equal_to`.",
			Optional:            true,
		},
		"occurrence_low": schema.Int64Attribute{
			MarkdownDescription: "Lower occurrence bound when operator is `between`.",
			Optional:            true,
		},
		"occurrence_high": schema.Int64Attribute{
			MarkdownDescription: "Upper occurrence bound when operator is `between`.",
			Optional:            true,
		},
		"confidence_level": schema.StringAttribute{
			MarkdownDescription: "Confidence level: `low`, `medium`, `high`. Defaults to `high`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("high"),
			Validators: []validator.String{
				stringvalidator.OneOf("low", "medium", "high"),
			},
		},
		"match_type": schema.StringAttribute{
			MarkdownDescription: "Match type: `include` or `exclude`. Defaults to `include`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("include"),
			Validators: []validator.String{
				stringvalidator.OneOf("include", "exclude"),
			},
		},
		"by_unique_count": schema.BoolAttribute{
			MarkdownDescription: "Whether to evaluate occurrences by unique matches. Defaults to `false`.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
	}
}

// RuleItemDataSourceAttributes returns schema attributes for rule_item in data sources.
func RuleItemDataSourceAttributes() map[string]dsschema.Attribute {
	return map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			MarkdownDescription: "The ID of the detection primitive.",
			Computed:            true,
		},
		"name": dsschema.StringAttribute{
			MarkdownDescription: "Descriptive name of the rule item.",
			Computed:            true,
		},
		"detection_technique": dsschema.StringAttribute{
			MarkdownDescription: "The detection technique used.",
			Computed:            true,
		},
		"occurrence_operator_type": dsschema.StringAttribute{
			MarkdownDescription: "Occurrence operator.",
			Computed:            true,
		},
		"occurrence_count": dsschema.Int64Attribute{
			MarkdownDescription: "Match count threshold.",
			Computed:            true,
		},
		"occurrence_low": dsschema.Int64Attribute{
			MarkdownDescription: "Lower occurrence bound.",
			Computed:            true,
		},
		"occurrence_high": dsschema.Int64Attribute{
			MarkdownDescription: "Upper occurrence bound.",
			Computed:            true,
		},
		"confidence_level": dsschema.StringAttribute{
			MarkdownDescription: "Confidence level.",
			Computed:            true,
		},
		"match_type": dsschema.StringAttribute{
			MarkdownDescription: "Match type.",
			Computed:            true,
		},
		"by_unique_count": dsschema.BoolAttribute{
			MarkdownDescription: "Whether to evaluate occurrences by unique matches.",
			Computed:            true,
		},
	}
}

// DataProfileResourceSchema defines the schema for scm_dlp_data_profile resource.
var DataProfileResourceSchema = schema.Schema{
	MarkdownDescription: "DLP Data Profile resource (`/v2/api/data-profiles`). Manages detection rules with boolean expression trees or multi-profile compositions.",
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The unique identifier of the data profile.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"tfid": schema.StringAttribute{
			MarkdownDescription: "The Terraform ID.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the data profile (up to 64 characters).",
			Required:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "Description of the data profile.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"profile_type": schema.StringAttribute{
			MarkdownDescription: "Profile type: `advanced` or `basic`. Defaults to `advanced`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("advanced"),
		},
		"profile_status": schema.StringAttribute{
			MarkdownDescription: "Profile status: `active` or `inactive`. Defaults to `active`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("active"),
		},
		"is_granular_data_profile": schema.BoolAttribute{
			MarkdownDescription: "Whether this is a granular data profile composing multiple child data profiles.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"version": schema.Int64Attribute{
			MarkdownDescription: "The version number of the profile.",
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"detection_rules": schema.ListNestedAttribute{
			MarkdownDescription: "List of detection rules. Must be all `expression_tree` or all `multi_profile`.",
			Required:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"rule_type": schema.StringAttribute{
						MarkdownDescription: "Rule type: `expression_tree` or `multi_profile`.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("expression_tree", "multi_profile"),
						},
					},
					"multi_profile": schema.SingleNestedAttribute{
						MarkdownDescription: "Multi-profile composition rule (combines multiple data profile IDs).",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"operator_type": schema.StringAttribute{
								MarkdownDescription: "Operator type: must be `or`.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("or"),
								},
							},
							"data_profile_ids": schema.ListAttribute{
								MarkdownDescription: "List of child data profile IDs.",
								ElementType:         types.Int64Type,
								Required:            true,
							},
						},
					},
					"expression_tree": schema.SingleNestedAttribute{
						MarkdownDescription: "Expression tree defining boolean logic and leaf rule items.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"operator_type": schema.StringAttribute{
								MarkdownDescription: "Logical operator: `and`, `or`, `not`, `and_not`, `or_not`.",
								Optional:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("and", "or", "not", "and_not", "or_not"),
								},
							},
							"rule_item": schema.SingleNestedAttribute{
								MarkdownDescription: "Leaf detection rule item.",
								Optional:            true,
								Attributes:          RuleItemResourceAttributes(),
							},
							"expression_tree_json": schema.StringAttribute{
								MarkdownDescription: "Optional raw JSON escape hatch for arbitrarily deep expression trees.",
								Optional:            true,
								Computed:            true,
							},
							"sub_expressions": schema.ListNestedAttribute{
								MarkdownDescription: "Sub-expressions (level 1) under this logical operator.",
								Optional:            true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"operator_type": schema.StringAttribute{
											MarkdownDescription: "Logical operator: `and`, `or`, `not`, `and_not`, `or_not`.",
											Optional:            true,
											Validators: []validator.String{
												stringvalidator.OneOf("and", "or", "not", "and_not", "or_not"),
											},
										},
										"rule_item": schema.SingleNestedAttribute{
											MarkdownDescription: "Leaf detection rule item.",
											Optional:            true,
											Attributes:          RuleItemResourceAttributes(),
										},
										"sub_expressions": schema.ListNestedAttribute{
											MarkdownDescription: "Nested sub-expressions (level 2) enabling native HCL boolean structures like OR-of-ANDs.",
											Optional:            true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"operator_type": schema.StringAttribute{
														MarkdownDescription: "Logical operator: `and`, `or`, `not`, `and_not`, `or_not`.",
														Optional:            true,
														Validators: []validator.String{
															stringvalidator.OneOf("and", "or", "not", "and_not", "or_not"),
														},
													},
													"rule_item": schema.SingleNestedAttribute{
														MarkdownDescription: "Leaf detection rule item.",
														Optional:            true,
														Attributes:          RuleItemResourceAttributes(),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

// DataProfileDataSourceSchema defines the schema for scm_dlp_data_profile data source.
var DataProfileDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "DLP Data Profile data source (`/v2/api/data-profiles/{id}`).",
	Attributes: map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			MarkdownDescription: "The unique identifier of the data profile.",
			Required:            true,
		},
		"tfid": dsschema.StringAttribute{
			MarkdownDescription: "The Terraform ID.",
			Computed:            true,
		},
		"name": dsschema.StringAttribute{
			MarkdownDescription: "The name of the data profile.",
			Computed:            true,
		},
		"description": dsschema.StringAttribute{
			MarkdownDescription: "Description of the data profile.",
			Computed:            true,
		},
		"profile_type": dsschema.StringAttribute{
			MarkdownDescription: "Profile type (`advanced`, `basic`).",
			Computed:            true,
		},
		"profile_status": dsschema.StringAttribute{
			MarkdownDescription: "Profile status (`active`, `inactive`).",
			Computed:            true,
		},
		"is_granular_data_profile": dsschema.BoolAttribute{
			MarkdownDescription: "Whether this is a granular data profile.",
			Computed:            true,
		},
		"version": dsschema.Int64Attribute{
			MarkdownDescription: "The version number of the profile.",
			Computed:            true,
		},
		"detection_rules": dsschema.ListNestedAttribute{
			MarkdownDescription: "List of detection rules.",
			Computed:            true,
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"rule_type": dsschema.StringAttribute{
						MarkdownDescription: "Rule type.",
						Computed:            true,
					},
					"multi_profile": dsschema.SingleNestedAttribute{
						MarkdownDescription: "Multi-profile composition rule.",
						Computed:            true,
						Attributes: map[string]dsschema.Attribute{
							"operator_type": dsschema.StringAttribute{
								MarkdownDescription: "Operator type.",
								Computed:            true,
							},
							"data_profile_ids": dsschema.ListAttribute{
								MarkdownDescription: "List of child data profile IDs.",
								ElementType:         types.Int64Type,
								Computed:            true,
							},
						},
					},
					"expression_tree": dsschema.SingleNestedAttribute{
						MarkdownDescription: "Expression tree.",
						Computed:            true,
						Attributes: map[string]dsschema.Attribute{
							"operator_type": dsschema.StringAttribute{
								MarkdownDescription: "Logical operator.",
								Computed:            true,
							},
							"rule_item": schemaToDsSingleNested(RuleItemDataSourceAttributes()),
							"expression_tree_json": dsschema.StringAttribute{
								MarkdownDescription: "Raw JSON expression tree.",
								Computed:            true,
							},
							"sub_expressions": dsschema.ListNestedAttribute{
								MarkdownDescription: "Sub-expressions (level 1).",
								Computed:            true,
								NestedObject: dsschema.NestedAttributeObject{
									Attributes: map[string]dsschema.Attribute{
										"operator_type": dsschema.StringAttribute{
											MarkdownDescription: "Logical operator.",
											Computed:            true,
										},
										"rule_item": schemaToDsSingleNested(RuleItemDataSourceAttributes()),
										"sub_expressions": dsschema.ListNestedAttribute{
											MarkdownDescription: "Sub-expressions (level 2).",
											Computed:            true,
											NestedObject: dsschema.NestedAttributeObject{
												Attributes: map[string]dsschema.Attribute{
													"operator_type": dsschema.StringAttribute{
														MarkdownDescription: "Logical operator.",
														Computed:            true,
													},
													"rule_item": schemaToDsSingleNested(RuleItemDataSourceAttributes()),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

func schemaToDsSingleNested(attrs map[string]dsschema.Attribute) dsschema.SingleNestedAttribute {
	return dsschema.SingleNestedAttribute{
		MarkdownDescription: "Leaf rule item.",
		Computed:            true,
		Attributes:          attrs,
	}
}

// DataProfileListDataSourceModel maps the list data source.
type DataProfileListDataSourceModel struct {
	Tfid  types.String               `tfsdk:"tfid"`
	Name  types.String               `tfsdk:"name"`
	Total types.Int64                `tfsdk:"total"`
	Data  []DataProfileResourceModel `tfsdk:"data"`
}

// DataProfileListDataSourceSchema defines the schema for scm_dlp_data_profile_list data source.
var DataProfileListDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "Retrieves a list of DLP Data Profiles (`GET /v2/api/data-profiles`), optionally filtered by name.",
	Attributes: map[string]dsschema.Attribute{
		"tfid": dsschema.StringAttribute{
			MarkdownDescription: "The Terraform ID.",
			Computed:            true,
		},
		"name": dsschema.StringAttribute{
			MarkdownDescription: "Optional profile name filter to find a specific data profile.",
			Optional:            true,
		},
		"total": dsschema.Int64Attribute{
			MarkdownDescription: "Total number of profiles returned.",
			Computed:            true,
		},
		"data": dsschema.ListNestedAttribute{
			MarkdownDescription: "List of matching data profiles.",
			Computed:            true,
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"id": dsschema.StringAttribute{
						MarkdownDescription: "The unique identifier of the data profile.",
						Computed:            true,
					},
					"tfid": dsschema.StringAttribute{
						MarkdownDescription: "The Terraform ID.",
						Computed:            true,
					},
					"name": dsschema.StringAttribute{
						MarkdownDescription: "The name of the data profile.",
						Computed:            true,
					},
					"description": dsschema.StringAttribute{
						MarkdownDescription: "Description of the data profile.",
						Computed:            true,
					},
					"profile_type": dsschema.StringAttribute{
						MarkdownDescription: "Profile type.",
						Computed:            true,
					},
					"profile_status": dsschema.StringAttribute{
						MarkdownDescription: "Profile status.",
						Computed:            true,
					},
					"is_granular_data_profile": dsschema.BoolAttribute{
						MarkdownDescription: "Whether this is a granular data profile.",
						Computed:            true,
					},
					"version": dsschema.Int64Attribute{
						MarkdownDescription: "Version of the data profile.",
						Computed:            true,
					},
					"detection_rules": dsschema.ListNestedAttribute{
						MarkdownDescription: "List of detection rules.",
						Computed:            true,
						NestedObject: dsschema.NestedAttributeObject{
							Attributes: map[string]dsschema.Attribute{
								"rule_type": dsschema.StringAttribute{
									MarkdownDescription: "Rule type.",
									Computed:            true,
								},
								"multi_profile": dsschema.SingleNestedAttribute{
									MarkdownDescription: "Multi-profile rule.",
									Computed:            true,
									Attributes: map[string]dsschema.Attribute{
										"operator_type": dsschema.StringAttribute{
											MarkdownDescription: "Operator type.",
											Computed:            true,
										},
										"data_profile_ids": dsschema.ListAttribute{
											MarkdownDescription: "Data profile IDs.",
											ElementType:         types.Int64Type,
											Computed:            true,
										},
									},
								},
								"expression_tree": dsschema.SingleNestedAttribute{
									MarkdownDescription: "Expression tree.",
									Computed:            true,
									Attributes: map[string]dsschema.Attribute{
										"operator_type": dsschema.StringAttribute{
											MarkdownDescription: "Operator type.",
											Computed:            true,
										},
										"rule_item": schemaToDsSingleNested(RuleItemDataSourceAttributes()),
										"expression_tree_json": dsschema.StringAttribute{
											MarkdownDescription: "Raw JSON expression tree.",
											Computed:            true,
										},
										"sub_expressions": dsschema.ListNestedAttribute{
											MarkdownDescription: "Sub-expressions (level 1).",
											Computed:            true,
											NestedObject: dsschema.NestedAttributeObject{
												Attributes: map[string]dsschema.Attribute{
													"operator_type": dsschema.StringAttribute{
														MarkdownDescription: "Operator type.",
														Computed:            true,
													},
													"rule_item": schemaToDsSingleNested(RuleItemDataSourceAttributes()),
													"sub_expressions": dsschema.ListNestedAttribute{
														MarkdownDescription: "Sub-expressions (level 2).",
														Computed:            true,
														NestedObject: dsschema.NestedAttributeObject{
															Attributes: map[string]dsschema.Attribute{
																"operator_type": dsschema.StringAttribute{
																	MarkdownDescription: "Operator type.",
																	Computed:            true,
																},
																"rule_item": schemaToDsSingleNested(RuleItemDataSourceAttributes()),
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

// ---------------------------------------------------------------------------
// 2. Data Filtering Profile Models & Schemas
// ---------------------------------------------------------------------------

// AppExclusionModel represents an item in app_exclusion_list.
type AppExclusionModel struct {
	AppId   types.String `tfsdk:"app_id"`
	AppName types.String `tfsdk:"app_name"`
	Type    types.String `tfsdk:"type"`
}

func (m AppExclusionModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"app_id":   basetypes.StringType{},
		"app_name": basetypes.StringType{},
		"type":     basetypes.StringType{},
	}
}

// URLExclusionModel represents an item in url_exclusion_list.
type URLExclusionModel struct {
	UrlId   types.String `tfsdk:"url_id"`
	UrlName types.String `tfsdk:"url_name"`
	Type    types.String `tfsdk:"type"`
}

func (m URLExclusionModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"url_id":   basetypes.StringType{},
		"url_name": basetypes.StringType{},
		"type":     basetypes.StringType{},
	}
}

// ExceptionRuleModel represents an item in exception_rules.
type ExceptionRuleModel struct {
	Name        types.String `tfsdk:"name"`
	Action      types.String `tfsdk:"action"`
	LogSeverity types.String `tfsdk:"log_severity"`
}

func (m ExceptionRuleModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":         basetypes.StringType{},
		"action":       basetypes.StringType{},
		"log_severity": basetypes.StringType{},
	}
}

// DataFilteringProfileResourceModel is the top-level resource model for scm_dlp_data_filtering_profile.
type DataFilteringProfileResourceModel struct {
	Tfid                     types.String         `tfsdk:"tfid"`
	Id                       types.String         `tfsdk:"id"`
	DataProfileId            types.Int64          `tfsdk:"data_profile_id"`
	Name                     types.String         `tfsdk:"name"`
	Description              types.String         `tfsdk:"description"`
	Direction                types.String         `tfsdk:"direction"`
	ScanType                 types.String         `tfsdk:"scan_type"`
	LogSeverity              types.String         `tfsdk:"log_severity"`
	FileBased                types.Bool           `tfsdk:"file_based"`
	NonFileBased             types.Bool           `tfsdk:"non_file_based"`
	FileType                 []types.String       `tfsdk:"file_type"`
	IsEndUserCoachingEnabled types.Bool           `tfsdk:"is_end_user_coaching_enabled"`
	EucTemplateId            types.String         `tfsdk:"euc_template_id"`
	Rule1Action              types.String         `tfsdk:"rule1_action"`
	Rule2Action              types.String         `tfsdk:"rule2_action"`
	AppExclusionList         []AppExclusionModel  `tfsdk:"app_exclusion_list"`
	UrlExclusionList         []URLExclusionModel  `tfsdk:"url_exclusion_list"`
	ExceptionRules           []ExceptionRuleModel `tfsdk:"exception_rules"`
}

// DataFilteringProfileResourceSchema defines the schema for scm_dlp_data_filtering_profile resource.
var DataFilteringProfileResourceSchema = schema.Schema{
	MarkdownDescription: "DLP Data Filtering Profile resource (`/v2/api/data-filtering-profiles`). Manages scan directions, file type scopes, user coaching, exclusions, and severity actions linked 1:1 to a Data Profile.",
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The unique identifier of the data filtering profile. If omitted during creation, it is auto-resolved from the data_profile_id.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"tfid": schema.StringAttribute{
			MarkdownDescription: "The Terraform ID.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"data_profile_id": schema.Int64Attribute{
			MarkdownDescription: "The ID of the parent data profile this filtering profile is linked to.",
			Required:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the data filtering profile.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "Description of the data filtering profile.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"direction": schema.StringAttribute{
			MarkdownDescription: "Scan direction: `both`, `c2s` (upload), `s2c` (download).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("both"),
		},
		"scan_type": schema.StringAttribute{
			MarkdownDescription: "Scan type: `include` or `exclude`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("include"),
		},
		"log_severity": schema.StringAttribute{
			MarkdownDescription: "Log severity: `CRITICAL`, `HIGH`, `MEDIUM`, `LOW`, `INFORMATIONAL`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("HIGH"),
			Validators: []validator.String{
				stringvalidator.OneOf("CRITICAL", "HIGH", "MEDIUM", "LOW", "INFORMATIONAL", "critical", "high", "medium", "low", "informational"),
			},
		},
		"file_based": schema.BoolAttribute{
			MarkdownDescription: "Whether to scan file-based traffic.",
			Required:            true,
		},
		"non_file_based": schema.BoolAttribute{
			MarkdownDescription: "Whether to scan non-file-based traffic.",
			Required:            true,
		},
		"file_type": schema.ListAttribute{
			MarkdownDescription: "List of file types to scan or filter.",
			ElementType:         types.StringType,
			Optional:            true,
		},
		"is_end_user_coaching_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether End User Coaching (EUC) is enabled for this profile.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"euc_template_id": schema.StringAttribute{
			MarkdownDescription: "The End User Coaching template ID.",
			Optional:            true,
			Computed:            true,
		},
		"rule1_action": schema.StringAttribute{
			MarkdownDescription: "Action for rule 1 (e.g. `alert`, `block`, `allow`). Maps to `rule1.action` in the API.",
			Optional:            true,
			Computed:            true,
		},
		"rule2_action": schema.StringAttribute{
			MarkdownDescription: "Action for rule 2 (e.g. `alert`, `block`, `allow`). Maps to `rule2.action` in the API.",
			Optional:            true,
			Computed:            true,
		},
		"app_exclusion_list": schema.ListNestedAttribute{
			MarkdownDescription: "List of application exclusions.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"app_id": schema.StringAttribute{
						MarkdownDescription: "Application ID.",
						Required:            true,
					},
					"app_name": schema.StringAttribute{
						MarkdownDescription: "Application name.",
						Optional:            true,
						Computed:            true,
					},
					"type": schema.StringAttribute{
						MarkdownDescription: "Application type.",
						Optional:            true,
						Computed:            true,
					},
				},
			},
		},
		"url_exclusion_list": schema.ListNestedAttribute{
			MarkdownDescription: "List of URL exclusions.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"url_id": schema.StringAttribute{
						MarkdownDescription: "URL ID.",
						Required:            true,
					},
					"url_name": schema.StringAttribute{
						MarkdownDescription: "URL pattern or name.",
						Optional:            true,
						Computed:            true,
					},
					"type": schema.StringAttribute{
						MarkdownDescription: "URL type.",
						Optional:            true,
						Computed:            true,
					},
				},
			},
		},
		"exception_rules": schema.ListNestedAttribute{
			MarkdownDescription: "List of exception rules overriding standard actions.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Exception rule name.",
						Optional:            true,
					},
					"action": schema.StringAttribute{
						MarkdownDescription: "Exception rule action (e.g. `allow`, `alert`, `block`).",
						Optional:            true,
					},
					"log_severity": schema.StringAttribute{
						MarkdownDescription: "Exception rule log severity.",
						Optional:            true,
					},
				},
			},
		},
	},
}

// DataFilteringProfileDataSourceSchema defines the schema for scm_dlp_data_filtering_profile data source.
var DataFilteringProfileDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "DLP Data Filtering Profile data source (`/v2/api/data-filtering-profiles/{id}`).",
	Attributes: map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			MarkdownDescription: "The unique identifier of the data filtering profile.",
			Required:            true,
		},
		"tfid": dsschema.StringAttribute{
			MarkdownDescription: "The Terraform ID.",
			Computed:            true,
		},
		"data_profile_id": dsschema.Int64Attribute{
			MarkdownDescription: "The ID of the parent data profile.",
			Computed:            true,
		},
		"name": dsschema.StringAttribute{
			MarkdownDescription: "The name of the data filtering profile.",
			Computed:            true,
		},
		"description": dsschema.StringAttribute{
			MarkdownDescription: "Description of the data filtering profile.",
			Computed:            true,
		},
		"direction": dsschema.StringAttribute{
			MarkdownDescription: "Scan direction.",
			Computed:            true,
		},
		"scan_type": dsschema.StringAttribute{
			MarkdownDescription: "Scan type.",
			Computed:            true,
		},
		"log_severity": dsschema.StringAttribute{
			MarkdownDescription: "Log severity.",
			Computed:            true,
		},
		"file_based": dsschema.BoolAttribute{
			MarkdownDescription: "Whether file-based scanning is active.",
			Computed:            true,
		},
		"non_file_based": dsschema.BoolAttribute{
			MarkdownDescription: "Whether non-file-based scanning is active.",
			Computed:            true,
		},
		"file_type": dsschema.ListAttribute{
			MarkdownDescription: "List of file types.",
			ElementType:         types.StringType,
			Computed:            true,
		},
		"is_end_user_coaching_enabled": dsschema.BoolAttribute{
			MarkdownDescription: "Whether End User Coaching is enabled.",
			Computed:            true,
		},
		"euc_template_id": dsschema.StringAttribute{
			MarkdownDescription: "End User Coaching template ID.",
			Computed:            true,
		},
		"rule1_action": dsschema.StringAttribute{
			MarkdownDescription: "Action for rule 1.",
			Computed:            true,
		},
		"rule2_action": dsschema.StringAttribute{
			MarkdownDescription: "Action for rule 2.",
			Computed:            true,
		},
		"app_exclusion_list": dsschema.ListNestedAttribute{
			MarkdownDescription: "List of application exclusions.",
			Computed:            true,
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"app_id": dsschema.StringAttribute{
						MarkdownDescription: "Application ID.",
						Computed:            true,
					},
					"app_name": dsschema.StringAttribute{
						MarkdownDescription: "Application name.",
						Computed:            true,
					},
					"type": dsschema.StringAttribute{
						MarkdownDescription: "Application type.",
						Computed:            true,
					},
				},
			},
		},
		"url_exclusion_list": dsschema.ListNestedAttribute{
			MarkdownDescription: "List of URL exclusions.",
			Computed:            true,
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"url_id": dsschema.StringAttribute{
						MarkdownDescription: "URL ID.",
						Computed:            true,
					},
					"url_name": dsschema.StringAttribute{
						MarkdownDescription: "URL pattern or name.",
						Computed:            true,
					},
					"type": dsschema.StringAttribute{
						MarkdownDescription: "URL type.",
						Computed:            true,
					},
				},
			},
		},
		"exception_rules": dsschema.ListNestedAttribute{
			MarkdownDescription: "List of exception rules.",
			Computed:            true,
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"name": dsschema.StringAttribute{
						MarkdownDescription: "Exception rule name.",
						Computed:            true,
					},
					"action": dsschema.StringAttribute{
						MarkdownDescription: "Exception rule action.",
						Computed:            true,
					},
					"log_severity": dsschema.StringAttribute{
						MarkdownDescription: "Exception rule log severity.",
						Computed:            true,
					},
				},
			},
		},
	},
}

// DataFilteringProfileListDataSourceModel maps the list data source.
type DataFilteringProfileListDataSourceModel struct {
	Tfid   types.String                        `tfsdk:"tfid"`
	Name   types.String                        `tfsdk:"name"`
	Status types.String                        `tfsdk:"status"`
	Total  types.Int64                         `tfsdk:"total"`
	Data   []DataFilteringProfileResourceModel `tfsdk:"data"`
}

// DataFilteringProfileListDataSourceSchema defines the schema for scm_dlp_data_filtering_profile_list data source.
var DataFilteringProfileListDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "Retrieves a list of DLP Data Filtering Profiles (`GET /v2/api/data-filtering-profiles`), optionally filtered by name or status.",
	Attributes: map[string]dsschema.Attribute{
		"tfid": dsschema.StringAttribute{
			MarkdownDescription: "The Terraform ID.",
			Computed:            true,
		},
		"name": dsschema.StringAttribute{
			MarkdownDescription: "Optional filtering profile name filter.",
			Optional:            true,
		},
		"status": dsschema.StringAttribute{
			MarkdownDescription: "Optional status filter.",
			Optional:            true,
		},
		"total": dsschema.Int64Attribute{
			MarkdownDescription: "Total number of filtering profiles returned.",
			Computed:            true,
		},
		"data": dsschema.ListNestedAttribute{
			MarkdownDescription: "List of matching data filtering profiles.",
			Computed:            true,
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"id": dsschema.StringAttribute{
						MarkdownDescription: "The unique identifier of the data filtering profile.",
						Computed:            true,
					},
					"tfid": dsschema.StringAttribute{
						MarkdownDescription: "The Terraform ID.",
						Computed:            true,
					},
					"data_profile_id": dsschema.Int64Attribute{
						MarkdownDescription: "The ID of the parent data profile.",
						Computed:            true,
					},
					"name": dsschema.StringAttribute{
						MarkdownDescription: "The name of the data filtering profile.",
						Computed:            true,
					},
					"description": dsschema.StringAttribute{
						MarkdownDescription: "Description of the data filtering profile.",
						Computed:            true,
					},
					"direction": dsschema.StringAttribute{
						MarkdownDescription: "Scan direction.",
						Computed:            true,
					},
					"scan_type": dsschema.StringAttribute{
						MarkdownDescription: "Scan type.",
						Computed:            true,
					},
					"log_severity": dsschema.StringAttribute{
						MarkdownDescription: "Log severity.",
						Computed:            true,
					},
					"file_based": dsschema.BoolAttribute{
						MarkdownDescription: "Whether file-based scanning is active.",
						Computed:            true,
					},
					"non_file_based": dsschema.BoolAttribute{
						MarkdownDescription: "Whether non-file-based scanning is active.",
						Computed:            true,
					},
					"file_type": dsschema.ListAttribute{
						MarkdownDescription: "List of file types.",
						ElementType:         types.StringType,
						Computed:            true,
					},
					"is_end_user_coaching_enabled": dsschema.BoolAttribute{
						MarkdownDescription: "Whether End User Coaching is enabled.",
						Computed:            true,
					},
					"euc_template_id": dsschema.StringAttribute{
						MarkdownDescription: "End User Coaching template ID.",
						Computed:            true,
					},
					"rule1_action": dsschema.StringAttribute{
						MarkdownDescription: "Action for rule 1.",
						Computed:            true,
					},
					"rule2_action": dsschema.StringAttribute{
						MarkdownDescription: "Action for rule 2.",
						Computed:            true,
					},
					"app_exclusion_list": dsschema.ListNestedAttribute{
						MarkdownDescription: "List of application exclusions.",
						Computed:            true,
						NestedObject: dsschema.NestedAttributeObject{
							Attributes: map[string]dsschema.Attribute{
								"app_id": dsschema.StringAttribute{
									MarkdownDescription: "Application ID.",
									Computed:            true,
								},
								"app_name": dsschema.StringAttribute{
									MarkdownDescription: "Application name.",
									Computed:            true,
								},
								"type": dsschema.StringAttribute{
									MarkdownDescription: "Application type.",
									Computed:            true,
								},
							},
						},
					},
					"url_exclusion_list": dsschema.ListNestedAttribute{
						MarkdownDescription: "List of URL exclusions.",
						Computed:            true,
						NestedObject: dsschema.NestedAttributeObject{
							Attributes: map[string]dsschema.Attribute{
								"url_id": dsschema.StringAttribute{
									MarkdownDescription: "URL ID.",
									Computed:            true,
								},
								"url_name": dsschema.StringAttribute{
									MarkdownDescription: "URL pattern or name.",
									Computed:            true,
								},
								"type": dsschema.StringAttribute{
									MarkdownDescription: "URL type.",
									Computed:            true,
								},
							},
						},
					},
					"exception_rules": dsschema.ListNestedAttribute{
						MarkdownDescription: "List of exception rules.",
						Computed:            true,
						NestedObject: dsschema.NestedAttributeObject{
							Attributes: map[string]dsschema.Attribute{
								"name": dsschema.StringAttribute{
									MarkdownDescription: "Exception rule name.",
									Computed:            true,
								},
								"action": dsschema.StringAttribute{
									MarkdownDescription: "Exception rule action.",
									Computed:            true,
								},
								"log_severity": dsschema.StringAttribute{
									MarkdownDescription: "Exception rule log severity.",
									Computed:            true,
								},
							},
						},
					},
				},
			},
		},
	},
}
