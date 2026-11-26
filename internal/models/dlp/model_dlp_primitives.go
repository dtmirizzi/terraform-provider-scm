package models

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ---------------------------------------------------------------------------
// 1. Data Pattern
// ---------------------------------------------------------------------------

// DataPatternRegexRuleModel represents a single regex rule nested in matching_rules.
type DataPatternRegexRuleModel struct {
	Regex  types.String  `tfsdk:"regex"`
	Weight types.Float64 `tfsdk:"weight"`
}

func (m DataPatternRegexRuleModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"regex":  basetypes.StringType{},
		"weight": basetypes.Float64Type{},
	}
}

// DataPatternMatchingRulesModel represents matching_rules configuration.
type DataPatternMatchingRulesModel struct {
	ProximityDistance types.Int64                 `tfsdk:"proximity_distance"`
	ProximityKeywords types.List                  `tfsdk:"proximity_keywords"` // types.List of types.String
	Regexes           []DataPatternRegexRuleModel `tfsdk:"regexes"`
}

func (m DataPatternMatchingRulesModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"proximity_distance": basetypes.Int64Type{},
		"proximity_keywords": basetypes.ListType{ElemType: basetypes.StringType{}},
		"regexes": basetypes.ListType{
			ElemType: basetypes.ObjectType{
				AttrTypes: DataPatternRegexRuleModel{}.AttrTypes(),
			},
		},
	}
}

// DataPatternDetectionConfigModel represents detection_config configuration.
type DataPatternDetectionConfigModel struct {
	Technique                 types.String `tfsdk:"technique"`
	SupportedConfidenceLevels types.List   `tfsdk:"supported_confidence_levels"` // types.List of types.String
}

func (m DataPatternDetectionConfigModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"technique":                   basetypes.StringType{},
		"supported_confidence_levels": basetypes.ListType{ElemType: basetypes.StringType{}},
	}
}

// DataPatternModel represents the Terraform model for scm_dlp_data_pattern.
type DataPatternModel struct {
	Tfid            types.String                     `tfsdk:"tfid"`
	Id              types.String                     `tfsdk:"id"`
	Name            types.String                     `tfsdk:"name"`
	Description     types.String                     `tfsdk:"description"`
	Type            types.String                     `tfsdk:"type"`
	Status          types.String                     `tfsdk:"status"`
	Version         types.Int64                      `tfsdk:"version"`
	DetectionConfig *DataPatternDetectionConfigModel `tfsdk:"detection_config"`
	MatchingRules   *DataPatternMatchingRulesModel   `tfsdk:"matching_rules"`
}

func (m DataPatternModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"tfid":        basetypes.StringType{},
		"id":          basetypes.StringType{},
		"name":        basetypes.StringType{},
		"description": basetypes.StringType{},
		"type":        basetypes.StringType{},
		"status":      basetypes.StringType{},
		"version":     basetypes.Int64Type{},
		"detection_config": basetypes.ObjectType{
			AttrTypes: DataPatternDetectionConfigModel{}.AttrTypes(),
		},
		"matching_rules": basetypes.ObjectType{
			AttrTypes: DataPatternMatchingRulesModel{}.AttrTypes(),
		},
	}
}

// DataPatternListModel represents the data source model for scm_dlp_data_pattern_list.
type DataPatternListModel struct {
	Name  types.String       `tfsdk:"name"`
	Total types.Int64        `tfsdk:"total"`
	Data  []DataPatternModel `tfsdk:"data"`
}

// DataPatternResourceSchema defines the schema for scm_dlp_data_pattern resource.
var DataPatternResourceSchema = schema.Schema{
	MarkdownDescription: "Manages an Enterprise DLP Data Pattern.",
	Attributes: map[string]schema.Attribute{
		"tfid": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The Terraform ID.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The unique identifier of the data pattern.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The name of the data pattern.",
		},
		"description": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "A description of the data pattern.",
		},
		"type": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The type of data pattern: custom, predefined, or file_property.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"status": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The status of the data pattern.",
		},
		"version": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "The version of the data pattern.",
		},
		"detection_config": schema.SingleNestedAttribute{
			Required:            true,
			MarkdownDescription: "Detection technique configuration.",
			Attributes: map[string]schema.Attribute{
				"technique": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "Detection technique (e.g. weighted_regex, regex, ml, etc.).",
				},
				"supported_confidence_levels": schema.ListAttribute{
					ElementType:         types.StringType,
					Required:            true,
					MarkdownDescription: "Supported confidence levels (low, medium, high).",
				},
			},
		},
		"matching_rules": schema.SingleNestedAttribute{
			Optional:            true,
			MarkdownDescription: "Matching rules for regex and weighted_regex techniques.",
			Attributes: map[string]schema.Attribute{
				"proximity_distance": schema.Int64Attribute{
					Optional:            true,
					Computed:            true,
					MarkdownDescription: "Proximity distance for keywords (2 to 1000).",
				},
				"proximity_keywords": schema.ListAttribute{
					ElementType:         types.StringType,
					Optional:            true,
					Computed:            true,
					MarkdownDescription: "List of proximity keywords.",
				},
				"regexes": schema.ListNestedAttribute{
					Optional:            true,
					Computed:            true,
					MarkdownDescription: "List of regex rules with weights.",
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"regex": schema.StringAttribute{
								Required:            true,
								MarkdownDescription: "Regular expression pattern.",
							},
							"weight": schema.Float64Attribute{
								Required:            true,
								MarkdownDescription: "Weight for the regular expression match.",
							},
						},
					},
				},
			},
		},
	},
}

// DataPatternDataSourceSchema defines the schema for scm_dlp_data_pattern data source.
var DataPatternDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "Retrieves details for a specific Enterprise DLP Data Pattern.",
	Attributes: map[string]dsschema.Attribute{
		"tfid": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The Terraform ID.",
		},
		"id": dsschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The unique identifier of the data pattern.",
		},
		"name": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The name of the data pattern.",
		},
		"description": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "A description of the data pattern.",
		},
		"type": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The type of data pattern: custom, predefined, or file_property.",
		},
		"status": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The status of the data pattern.",
		},
		"version": dsschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "The version of the data pattern.",
		},
		"detection_config": dsschema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Detection technique configuration.",
			Attributes: map[string]dsschema.Attribute{
				"technique": dsschema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Detection technique.",
				},
				"supported_confidence_levels": dsschema.ListAttribute{
					ElementType:         types.StringType,
					Computed:            true,
					MarkdownDescription: "Supported confidence levels.",
				},
			},
		},
		"matching_rules": dsschema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Matching rules for regex and weighted_regex techniques.",
			Attributes: map[string]dsschema.Attribute{
				"proximity_distance": dsschema.Int64Attribute{
					Computed:            true,
					MarkdownDescription: "Proximity distance for keywords.",
				},
				"proximity_keywords": dsschema.ListAttribute{
					ElementType:         types.StringType,
					Computed:            true,
					MarkdownDescription: "List of proximity keywords.",
				},
				"regexes": dsschema.ListNestedAttribute{
					Computed:            true,
					MarkdownDescription: "List of regex rules with weights.",
					NestedObject: dsschema.NestedAttributeObject{
						Attributes: map[string]dsschema.Attribute{
							"regex": dsschema.StringAttribute{
								Computed:            true,
								MarkdownDescription: "Regular expression pattern.",
							},
							"weight": dsschema.Float64Attribute{
								Computed:            true,
								MarkdownDescription: "Weight for the regular expression match.",
							},
						},
					},
				},
			},
		},
	},
}

// DataPatternListDataSourceSchema defines the schema for scm_dlp_data_pattern_list data source.
var DataPatternListDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "Lists Enterprise DLP Data Patterns with optional name filtering.",
	Attributes: map[string]dsschema.Attribute{
		"name": dsschema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Filter data patterns by name.",
		},
		"total": dsschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Total number of data patterns matching query.",
		},
		"data": dsschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "List of matching data patterns.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"tfid": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The Terraform ID.",
					},
					"id": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The unique identifier of the data pattern.",
					},
					"name": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The name of the data pattern.",
					},
					"description": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "A description of the data pattern.",
					},
					"type": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The type of data pattern.",
					},
					"status": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The status of the data pattern.",
					},
					"version": dsschema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "The version of the data pattern.",
					},
					"detection_config": dsschema.SingleNestedAttribute{
						Computed:            true,
						MarkdownDescription: "Detection technique configuration.",
						Attributes: map[string]dsschema.Attribute{
							"technique": dsschema.StringAttribute{
								Computed:            true,
								MarkdownDescription: "Detection technique.",
							},
							"supported_confidence_levels": dsschema.ListAttribute{
								ElementType:         types.StringType,
								Computed:            true,
								MarkdownDescription: "Supported confidence levels.",
							},
						},
					},
					"matching_rules": dsschema.SingleNestedAttribute{
						Computed:            true,
						MarkdownDescription: "Matching rules for regex and weighted_regex techniques.",
						Attributes: map[string]dsschema.Attribute{
							"proximity_distance": dsschema.Int64Attribute{
								Computed:            true,
								MarkdownDescription: "Proximity distance for keywords.",
							},
							"proximity_keywords": dsschema.ListAttribute{
								ElementType:         types.StringType,
								Computed:            true,
								MarkdownDescription: "List of proximity keywords.",
							},
							"regexes": dsschema.ListNestedAttribute{
								Computed:            true,
								MarkdownDescription: "List of regex rules with weights.",
								NestedObject: dsschema.NestedAttributeObject{
									Attributes: map[string]dsschema.Attribute{
										"regex": dsschema.StringAttribute{
											Computed:            true,
											MarkdownDescription: "Regular expression pattern.",
										},
										"weight": dsschema.Float64Attribute{
											Computed:            true,
											MarkdownDescription: "Weight for the regular expression match.",
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
// 2. Dictionary
// ---------------------------------------------------------------------------

// DictionaryModel represents the Terraform model for scm_dlp_dictionary.
type DictionaryModel struct {
	Tfid             types.String `tfsdk:"tfid"`
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Category         types.String `tfsdk:"category"`
	OriginalFileName types.String `tfsdk:"original_file_name"`
	RegionName       types.String `tfsdk:"region_name"`
	IsCaseSensitive  types.Bool   `tfsdk:"is_case_sensitive"`
	KeywordList      types.List   `tfsdk:"keyword_list"` // types.List of types.String
	KeywordCount     types.Int64  `tfsdk:"keyword_count"`
}

func (m DictionaryModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"tfid":               basetypes.StringType{},
		"id":                 basetypes.StringType{},
		"name":               basetypes.StringType{},
		"description":        basetypes.StringType{},
		"category":           basetypes.StringType{},
		"original_file_name": basetypes.StringType{},
		"region_name":        basetypes.StringType{},
		"is_case_sensitive":  basetypes.BoolType{},
		"keyword_list":       basetypes.ListType{ElemType: basetypes.StringType{}},
		"keyword_count":      basetypes.Int64Type{},
	}
}

// DictionaryListModel represents the data source model for scm_dlp_dictionary_list.
type DictionaryListModel struct {
	Total types.Int64       `tfsdk:"total"`
	Data  []DictionaryModel `tfsdk:"data"`
}

// DictionaryResourceSchema defines the schema for scm_dlp_dictionary resource.
var DictionaryResourceSchema = schema.Schema{
	MarkdownDescription: "Manages an Enterprise DLP Keyword Dictionary.",
	Attributes: map[string]schema.Attribute{
		"tfid": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The Terraform ID.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The unique identifier of the dictionary.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The name of the dictionary.",
		},
		"description": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "A description of the dictionary.",
		},
		"category": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Category of the dictionary (e.g., Confidential, Financial, Healthcare, etc.).",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"original_file_name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Original file name of the dictionary file (.txt or .csv).",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"region_name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Region for the dictionary (e.g. us, eu).",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"is_case_sensitive": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
			MarkdownDescription: "Whether keyword matching is case sensitive.",
		},
		"keyword_list": schema.ListAttribute{
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "List of keywords belonging to this dictionary. When omitted, the existing keywords are kept on update. Keywords must not have leading or trailing whitespace.",
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
			Validators: []validator.List{
				listvalidator.ValueStringsAre(
					stringvalidator.RegexMatches(regexp.MustCompile(`^\S(.*\S)?$`), "must not be empty or have leading or trailing whitespace"),
				),
			},
		},
		"keyword_count": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Total number of keywords in the dictionary.",
		},
	},
}

// DictionaryDataSourceSchema defines the schema for scm_dlp_dictionary data source.
var DictionaryDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "Retrieves details for an Enterprise DLP Keyword Dictionary.",
	Attributes: map[string]dsschema.Attribute{
		"tfid": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The Terraform ID.",
		},
		"id": dsschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The unique identifier of the dictionary.",
		},
		"name": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The name of the dictionary.",
		},
		"description": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "A description of the dictionary.",
		},
		"category": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Category of the dictionary.",
		},
		"original_file_name": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Original file name.",
		},
		"region_name": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Region name.",
		},
		"is_case_sensitive": dsschema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether keyword matching is case sensitive.",
		},
		"keyword_list": dsschema.ListAttribute{
			ElementType:         types.StringType,
			Computed:            true,
			MarkdownDescription: "List of keywords.",
		},
		"keyword_count": dsschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Total number of keywords.",
		},
	},
}

// DictionaryListDataSourceSchema defines the schema for scm_dlp_dictionary_list data source.
var DictionaryListDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "Lists Enterprise DLP Keyword Dictionaries.",
	Attributes: map[string]dsschema.Attribute{
		"total": dsschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Total number of dictionaries.",
		},
		"data": dsschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "List of dictionaries.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"tfid": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The Terraform ID.",
					},
					"id": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The unique identifier of the dictionary.",
					},
					"name": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The name of the dictionary.",
					},
					"description": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "A description of the dictionary.",
					},
					"category": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Category of the dictionary.",
					},
					"original_file_name": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Original file name.",
					},
					"region_name": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Region name.",
					},
					"is_case_sensitive": dsschema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether keyword matching is case sensitive.",
					},
					"keyword_list": dsschema.ListAttribute{
						ElementType:         types.StringType,
						Computed:            true,
						MarkdownDescription: "List of keywords.",
					},
					"keyword_count": dsschema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Total number of keywords.",
					},
				},
			},
		},
	},
}

// ---------------------------------------------------------------------------
// 3. Document Type
// ---------------------------------------------------------------------------

// DocumentTypeModel represents the Terraform model for scm_dlp_document_type.
type DocumentTypeModel struct {
	Tfid               types.String `tfsdk:"tfid"`
	Id                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Category           types.String `tfsdk:"category"`
	DetectionTechnique types.String `tfsdk:"detection_technique"`
	OriginalFileName   types.String `tfsdk:"original_file_name"`
	L1CategoryID       types.Int64  `tfsdk:"l1_category_id"`
	L2CategoryID       types.Int64  `tfsdk:"l2_category_id"`
}

func (m DocumentTypeModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"tfid":                basetypes.StringType{},
		"id":                  basetypes.StringType{},
		"name":                basetypes.StringType{},
		"description":         basetypes.StringType{},
		"category":            basetypes.StringType{},
		"detection_technique": basetypes.StringType{},
		"original_file_name":  basetypes.StringType{},
		"l1_category_id":      basetypes.Int64Type{},
		"l2_category_id":      basetypes.Int64Type{},
	}
}

// DocumentTypeListModel represents the data source model for scm_dlp_document_type_list.
type DocumentTypeListModel struct {
	Total types.Int64         `tfsdk:"total"`
	Data  []DocumentTypeModel `tfsdk:"data"`
}

// DocumentTypeResourceSchema defines the schema for scm_dlp_document_type resource.
var DocumentTypeResourceSchema = schema.Schema{
	MarkdownDescription: "Manages an Enterprise DLP Document Type.",
	Attributes: map[string]schema.Attribute{
		"tfid": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The Terraform ID.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The unique identifier of the document type.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The name of the document type.",
		},
		"description": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "A description of the document type.",
		},
		"category": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Category of the document type (e.g. Confidential, Financial, etc.).",
		},
		"detection_technique": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Detection technique: document_fingerprint or trainable_classifier.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"original_file_name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Original sample file name.",
		},
		"l1_category_id": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Level 1 Category ID.",
		},
		"l2_category_id": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Level 2 Category ID.",
		},
	},
}

// DocumentTypeDataSourceSchema defines the schema for scm_dlp_document_type data source.
var DocumentTypeDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "Retrieves details for an Enterprise DLP Document Type.",
	Attributes: map[string]dsschema.Attribute{
		"tfid": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The Terraform ID.",
		},
		"id": dsschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The unique identifier of the document type.",
		},
		"name": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The name of the document type.",
		},
		"description": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "A description of the document type.",
		},
		"category": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Category of the document type.",
		},
		"detection_technique": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Detection technique.",
		},
		"original_file_name": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Original sample file name.",
		},
		"l1_category_id": dsschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Level 1 Category ID.",
		},
		"l2_category_id": dsschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Level 2 Category ID.",
		},
	},
}

// DocumentTypeListDataSourceSchema defines the schema for scm_dlp_document_type_list data source.
var DocumentTypeListDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "Lists Enterprise DLP Document Types.",
	Attributes: map[string]dsschema.Attribute{
		"total": dsschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Total number of document types.",
		},
		"data": dsschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "List of document types.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"tfid": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The Terraform ID.",
					},
					"id": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The unique identifier of the document type.",
					},
					"name": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The name of the document type.",
					},
					"description": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "A description of the document type.",
					},
					"category": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Category of the document type.",
					},
					"detection_technique": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Detection technique.",
					},
					"original_file_name": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Original sample file name.",
					},
					"l1_category_id": dsschema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Level 1 Category ID.",
					},
					"l2_category_id": dsschema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Level 2 Category ID.",
					},
				},
			},
		},
	},
}

// ---------------------------------------------------------------------------
// 4. EDM Dataset
// ---------------------------------------------------------------------------

// EdmFieldModel represents a field in an EDM dataset.
type EdmFieldModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func (m EdmFieldModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   basetypes.StringType{},
		"name": basetypes.StringType{},
		"type": basetypes.StringType{},
	}
}

// EdmDatasetModel represents the Terraform model for scm_dlp_edm_dataset.
type EdmDatasetModel struct {
	Tfid           types.String    `tfsdk:"tfid"`
	Id             types.String    `tfsdk:"id"`
	Name           types.String    `tfsdk:"name"`
	RegionName     types.String    `tfsdk:"region_name"`
	TenantId       types.String    `tfsdk:"tenant_id"`
	IndexStatus    types.String    `tfsdk:"index_status"`
	IndexErrorCode types.String    `tfsdk:"index_error_code"`
	ActiveFields   []EdmFieldModel `tfsdk:"active_fields"`
	AllFields      []EdmFieldModel `tfsdk:"all_fields"`
}

func (m EdmDatasetModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"tfid":             basetypes.StringType{},
		"id":               basetypes.StringType{},
		"name":             basetypes.StringType{},
		"region_name":      basetypes.StringType{},
		"tenant_id":        basetypes.StringType{},
		"index_status":     basetypes.StringType{},
		"index_error_code": basetypes.StringType{},
		"active_fields": basetypes.ListType{
			ElemType: basetypes.ObjectType{
				AttrTypes: EdmFieldModel{}.AttrTypes(),
			},
		},
		"all_fields": basetypes.ListType{
			ElemType: basetypes.ObjectType{
				AttrTypes: EdmFieldModel{}.AttrTypes(),
			},
		},
	}
}

// EdmDatasetListModel represents the data source model for scm_dlp_edm_dataset_list.
type EdmDatasetListModel struct {
	Name  types.String      `tfsdk:"name"`
	Total types.Int64       `tfsdk:"total"`
	Data  []EdmDatasetModel `tfsdk:"data"`
}

// EdmDatasetResourceSchema defines the schema for scm_dlp_edm_dataset resource.
var EdmDatasetResourceSchema = schema.Schema{
	MarkdownDescription: "Adopts an existing Enterprise DLP Exact Data Match (EDM) Dataset into Terraform state. EDM datasets are created and indexed out-of-band with the EDM CLI, so this resource never creates or deletes data: `terraform destroy` only removes it from state.",
	Attributes: map[string]schema.Attribute{
		"tfid": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The Terraform ID.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "The unique identifier of the EDM dataset. If provided, the existing dataset is adopted into state; otherwise it is looked up by `name`.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplaceIf(func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
					// Re-adopt when the user points at a different dataset; ignore id becoming computed.
					resp.RequiresReplace = !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() && !req.StateValue.IsNull() && req.ConfigValue.ValueString() != req.StateValue.ValueString()
				}, "Adopting a different EDM dataset requires replacement.", "Adopting a different EDM dataset requires replacement."),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The name of the EDM dataset.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"region_name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Region where the dataset is stored.",
		},
		"tenant_id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The tenant ID.",
		},
		"index_status": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The index status of the dataset (e.g. COMPLETED, IN_PROGRESS, FAILED).",
		},
		"index_error_code": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Error code if indexing failed.",
		},
		"active_fields": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Active schema fields for the EDM dataset.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field identifier.",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field name.",
					},
					"type": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field type.",
					},
				},
			},
		},
		"all_fields": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "All schema fields for the EDM dataset.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field identifier.",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field name.",
					},
					"type": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field type.",
					},
				},
			},
		},
	},
}

// EdmDatasetDataSourceSchema defines the schema for scm_dlp_edm_dataset data source.
var EdmDatasetDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "Retrieves details for an Enterprise DLP Exact Data Match (EDM) Dataset.",
	Attributes: map[string]dsschema.Attribute{
		"tfid": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The Terraform ID.",
		},
		"id": dsschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The unique identifier of the EDM dataset.",
		},
		"name": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The name of the EDM dataset.",
		},
		"region_name": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Region where the dataset is stored.",
		},
		"tenant_id": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The tenant ID.",
		},
		"index_status": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The index status of the dataset.",
		},
		"index_error_code": dsschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Error code if indexing failed.",
		},
		"active_fields": dsschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Active schema fields for the EDM dataset.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"id": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field identifier.",
					},
					"name": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field name.",
					},
					"type": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field type.",
					},
				},
			},
		},
		"all_fields": dsschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "All schema fields for the EDM dataset.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"id": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field identifier.",
					},
					"name": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field name.",
					},
					"type": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Field type.",
					},
				},
			},
		},
	},
}

// EdmDatasetListDataSourceSchema defines the schema for scm_dlp_edm_dataset_list data source.
var EdmDatasetListDataSourceSchema = dsschema.Schema{
	MarkdownDescription: "Lists Enterprise DLP Exact Data Match (EDM) Datasets with optional name filtering.",
	Attributes: map[string]dsschema.Attribute{
		"name": dsschema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Filter datasets by name.",
		},
		"total": dsschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Total number of datasets matching query.",
		},
		"data": dsschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "List of matching EDM datasets.",
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"tfid": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The Terraform ID.",
					},
					"id": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The unique identifier of the EDM dataset.",
					},
					"name": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The name of the EDM dataset.",
					},
					"region_name": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Region where the dataset is stored.",
					},
					"tenant_id": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The tenant ID.",
					},
					"index_status": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The index status of the dataset.",
					},
					"index_error_code": dsschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Error code if indexing failed.",
					},
					"active_fields": dsschema.ListNestedAttribute{
						Computed:            true,
						MarkdownDescription: "Active schema fields for the EDM dataset.",
						NestedObject: dsschema.NestedAttributeObject{
							Attributes: map[string]dsschema.Attribute{
								"id": dsschema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Field identifier.",
								},
								"name": dsschema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Field name.",
								},
								"type": dsschema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Field type.",
								},
							},
						},
					},
					"all_fields": dsschema.ListNestedAttribute{
						Computed:            true,
						MarkdownDescription: "All schema fields for the EDM dataset.",
						NestedObject: dsschema.NestedAttributeObject{
							Attributes: map[string]dsschema.Attribute{
								"id": dsschema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Field identifier.",
								},
								"name": dsschema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Field name.",
								},
								"type": dsschema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Field type.",
								},
							},
						},
					},
				},
			},
		},
	},
}
