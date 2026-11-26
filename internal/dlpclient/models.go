package dlpclient

import "encoding/json"

// AuditResponse represents audit metadata returned by DLP v2 resources.
type AuditResponse struct {
	CreatedBy *string `json:"created_by,omitempty"`
	UpdatedBy *string `json:"updated_by,omitempty"`
	CreatedAt any     `json:"created_at,omitempty"`
	UpdatedAt any     `json:"updated_at,omitempty"`
}

// ---------------------------------------------------------------------------
// 1. Tenant Configuration, Snippets, OCR, Alerts, Channels
// ---------------------------------------------------------------------------

// TenantCommonConfigDTO maps /v1/api/tenant-common-config/{properties,update} and /v1/api/edm-dataset/{status,enable}.
type TenantCommonConfigDTO struct {
	ID                                     *string `json:"id,omitempty"`
	EnableSnippets                         *bool   `json:"enable_snippets,omitempty"`
	MaskLevel                              *string `json:"mask_level,omitempty"`
	OcrEnablement                          *string `json:"ocr_enablement,omitempty"`
	OcrEnablementBanner                    *bool   `json:"ocr_enablement_banner,omitempty"`
	EdmEnablement                          *string `json:"edm_enablement,omitempty"`
	EdmEnablementBanner                    *bool   `json:"edm_enablement_banner,omitempty"`
	ServiceName                            *string `json:"service_name,omitempty"`
	IncidentNotificationEnablement         *bool   `json:"incident_notification_enablement,omitempty"`
	IncidentNotificationSnippetsEnablement *bool   `json:"incident_notification_snippets_enablement,omitempty"`
	DetectionExemptionPeriodMinutes        *int64  `json:"detection_exemption_period_minutes,omitempty"`
	EnableStructuredDataClustering         *bool   `json:"enable_structured_data_clustering,omitempty"`
	EnforceStructuredDataClusteringUniform *bool   `json:"enforce_structured_data_clustering_uniformity,omitempty"`
	StructuredDataClusteringMinUniformity  *int64  `json:"structured_data_clustering_minimum_uniformity,omitempty"`
	AutoConfiguredProfileAndPatterns       *bool   `json:"auto_configured_profile_and_patterns,omitempty"`
	CreatedBy                              *string `json:"created_by,omitempty"`
	UpdatedBy                              *string `json:"updated_by,omitempty"`
}

// OcrEnablementDTO maps GET/PUT /v2/api/ocr.
type OcrEnablementDTO struct {
	OcrEnablement     string  `json:"ocr_enablement"`
	TargetServiceName string  `json:"target_service_name,omitempty"`
	ServiceName       string  `json:"service_name,omitempty"`
	TenantID          *string `json:"tenant_id,omitempty"`
}

// AlertSettingsDTO maps GET/PUT /v1/api/alerts/setting.
type AlertSettingsDTO struct {
	IncidentNotificationEnablement  *bool  `json:"incident_notification_enablement,omitempty"`
	DetectionExemptionPeriodMinutes *int64 `json:"detection_exemption_period_minutes,omitempty"`
}

// ---------------------------------------------------------------------------
// 2. Data Patterns (/v2/api/data-patterns)
// ---------------------------------------------------------------------------

type DataPatternDetectionConfig struct {
	Technique                 string   `json:"technique"`
	SupportedConfidenceLevels []string `json:"supported_confidence_levels,omitempty"`
}

type DataPatternRegexRule struct {
	Regex  string  `json:"regex"`
	Weight float64 `json:"weight"`
}

type DataPatternMatchingRules struct {
	Regexes           []DataPatternRegexRule `json:"regexes,omitempty"`
	ProximityDistance *int64                 `json:"proximity_distance,omitempty"`
	ProximityKeywords []string               `json:"proximity_keywords,omitempty"`
}

type DataPatternDTO struct {
	ID              *string                     `json:"id,omitempty"`
	Name            string                      `json:"name"`
	Description     *string                     `json:"description,omitempty"`
	TenantID        *string                     `json:"tenant_id,omitempty"`
	Status          *string                     `json:"status,omitempty"`
	Type            string                      `json:"type"`
	Version         *int64                      `json:"version,omitempty"`
	LicenseType     *string                     `json:"license_type,omitempty"`
	DetectionConfig *DataPatternDetectionConfig `json:"detection_config,omitempty"`
	MatchingRules   *DataPatternMatchingRules   `json:"matching_rules,omitempty"`
	IsParentManaged *bool                       `json:"is_parent_managed,omitempty"`
	AuditMetadata   *AuditResponse              `json:"audit_metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// 3. Dictionaries (/v2/api/dictionaries)
// ---------------------------------------------------------------------------

type DictionaryDTO struct {
	ID               *string        `json:"id,omitempty"`
	Name             string         `json:"name"`
	Description      *string        `json:"description,omitempty"`
	Category         string         `json:"category"`
	OriginalFileName string         `json:"original_file_name"`
	RegionName       string         `json:"region_name"`
	IsCaseSensitive  *bool          `json:"is_case_sensitive,omitempty"`
	Type             *string        `json:"type,omitempty"`
	TenantID         *string        `json:"tenant_id,omitempty"`
	Version          *int64         `json:"version,omitempty"`
	Keywords         []string       `json:"keywords,omitempty"`
	KeywordCount     *int64         `json:"keyword_count,omitempty"`
	IsParentManaged  *bool          `json:"is_parent_managed,omitempty"`
	AuditMetadata    *AuditResponse `json:"audit_metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// 4. Document Types (/v2/api/document-types)
// ---------------------------------------------------------------------------

type DocumentTypeDTO struct {
	ID                 *string        `json:"id,omitempty"`
	Name               string         `json:"name"`
	Description        *string        `json:"description,omitempty"`
	Category           string         `json:"category"`
	DetectionTechnique string         `json:"detection_technique"`
	OriginalFileName   string         `json:"original_file_name"`
	L1CategoryID       *int64         `json:"l1_category_id,omitempty"`
	L2CategoryID       *int64         `json:"l2_category_id,omitempty"`
	TenantID           *string        `json:"tenant_id,omitempty"`
	Version            *int64         `json:"version,omitempty"`
	IsParentManaged    *bool          `json:"is_parent_managed,omitempty"`
	AuditMetadata      *AuditResponse `json:"audit_metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// 5. EDM Datasets (/v2/api/edm-datasets)
// ---------------------------------------------------------------------------

type EdmFieldDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type EdmDatasetDTO struct {
	ID             *string        `json:"id,omitempty"`
	Name           string         `json:"name"`
	RegionName     *string        `json:"region_name,omitempty"`
	TenantID       *string        `json:"tenant_id,omitempty"`
	IndexStatus    *string        `json:"index_status,omitempty"`
	IndexErrorCode *string        `json:"index_error_code,omitempty"`
	ActiveFields   []EdmFieldDTO  `json:"active_fields,omitempty"`
	AllFields      []EdmFieldDTO  `json:"all_fields,omitempty"`
	AuditMetadata  *AuditResponse `json:"audit_metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// 6. Data Profiles (/v2/api/data-profiles)
// ---------------------------------------------------------------------------

type RuleItemDTO struct {
	ID                     string  `json:"id"`
	Name                   *string `json:"name,omitempty"`
	DetectionTechnique     string  `json:"detection_technique"`
	OccurrenceOperatorType string  `json:"occurrence_operator_type"`
	OccurrenceCount        *int64  `json:"occurrence_count,omitempty"`
	OccurrenceLow          *int64  `json:"occurrence_low,omitempty"`
	OccurrenceHigh         *int64  `json:"occurrence_high,omitempty"`
	ConfidenceLevel        *string `json:"confidence_level,omitempty"`
	MatchType              *string `json:"match_type,omitempty"`
	ByUniqueCount          *bool   `json:"by_unique_count,omitempty"`
	Version                *int64  `json:"version,omitempty"`
}

type ExpressionTreeNodeDTO struct {
	OperatorType   *string                 `json:"operator_type"`
	RuleItem       *RuleItemDTO            `json:"rule_item"`
	SubExpressions []ExpressionTreeNodeDTO `json:"sub_expressions"`
}

type MultiProfileNodeDTO struct {
	OperatorType   string  `json:"operator_type"`
	DataProfileIDs []int64 `json:"data_profile_ids"`
}

type DetectionRuleDTO struct {
	RuleType       string                 `json:"rule_type"`
	ExpressionTree *ExpressionTreeNodeDTO `json:"expression_tree,omitempty"`
	MultiProfile   *MultiProfileNodeDTO   `json:"multi_profile,omitempty"`
}

type DataProfileDTO struct {
	ID                    *string            `json:"id,omitempty"`
	Name                  string             `json:"name"`
	Description           *string            `json:"description,omitempty"`
	Version               *int64             `json:"version,omitempty"`
	TenantID              *string            `json:"tenant_id,omitempty"`
	ProfileType           *string            `json:"profile_type,omitempty"`
	ProfileStatus         *string            `json:"profile_status,omitempty"`
	Type                  *string            `json:"type,omitempty"`
	DetectionRules        []DetectionRuleDTO `json:"detection_rules"`
	IsParentManaged       *bool              `json:"is_parent_managed,omitempty"`
	IsGranularDataProfile *bool              `json:"is_granular_data_profile,omitempty"`
	AiOnboarding          *bool              `json:"ai_onboarding,omitempty"`
	AuditMetadata         *AuditResponse     `json:"audit_metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// 7. Data Filtering Profiles (/v2/api/data-filtering-profiles)
// ---------------------------------------------------------------------------

type DataFilteringRuleDTO struct {
	Action *string `json:"action,omitempty"`
}

type AppExclusionItemDTO struct {
	AppID   string  `json:"app_id"`
	AppName *string `json:"app_name,omitempty"`
	Type    *string `json:"type,omitempty"`
}

type URLExclusionItemDTO struct {
	URLID   string  `json:"url_id"`
	URLName *string `json:"url_name,omitempty"`
	Type    *string `json:"type,omitempty"`
}

type ExclusionsDTO struct {
	AppExclusionList []AppExclusionItemDTO `json:"app_exclusion_list,omitempty"`
	URLExclusionList []URLExclusionItemDTO `json:"url_exclusion_list,omitempty"`
}

type ExceptionRuleDTO struct {
	Name        *string `json:"name,omitempty"`
	Action      *string `json:"action,omitempty"`
	LogSeverity *string `json:"log_severity,omitempty"`
}

type DataFilteringProfileDTO struct {
	ID                       *string               `json:"id,omitempty"`
	DataProfileID            *int64                `json:"data_profile_id,omitempty"`
	Name                     *string               `json:"name,omitempty"`
	Description              *string               `json:"description,omitempty"`
	Direction                *string               `json:"direction,omitempty"`
	LogSeverity              *string               `json:"log_severity,omitempty"`
	ScanType                 *string               `json:"scan_type,omitempty"`
	FileBased                bool                  `json:"file_based"`
	NonFileBased             bool                  `json:"non_file_based"`
	FileType                 []string              `json:"file_type,omitempty"`
	Exclusions               *ExclusionsDTO        `json:"exclusions,omitempty"`
	Rule1                    *DataFilteringRuleDTO `json:"rule1,omitempty"`
	Rule2                    *DataFilteringRuleDTO `json:"rule2,omitempty"`
	ExceptionRules           []ExceptionRuleDTO    `json:"exception_rules,omitempty"`
	IsGranularProfile        *bool                 `json:"is_granular_profile,omitempty"`
	IsParentManaged          *bool                 `json:"is_parent_managed,omitempty"`
	IsEndUserCoachingEnabled *bool                 `json:"is_end_user_coaching_enabled,omitempty"`
	EucTemplateID            *string               `json:"euc_template_id,omitempty"`
	Version                  *int64                `json:"version,omitempty"`
	TenantID                 *string               `json:"tenant_id,omitempty"`
	AuditMetadata            *AuditResponse        `json:"audit_metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// 8. Endpoint Policies, Peripherals, Syslog, Log Forwarding, Config Export
// ---------------------------------------------------------------------------

type EndpointPolicyDTO struct {
	ID            *string         `json:"id,omitempty"`
	Name          string          `json:"name"`
	Description   *string         `json:"description,omitempty"`
	Enabled       bool            `json:"enabled"`
	Type          string          `json:"type"`
	Severity      string          `json:"severity"`
	Priority      *int64          `json:"priority,omitempty"`
	Scope         json.RawMessage `json:"scope"`
	Response      json.RawMessage `json:"response"`
	TenantID      *string         `json:"tenant_id,omitempty"`
	Version       *int64          `json:"version,omitempty"`
	AuditMetadata *AuditResponse  `json:"audit_metadata,omitempty"`
}

type PeripheralDTO struct {
	ID            *string        `json:"id,omitempty"`
	Name          string         `json:"name"`
	Description   *string        `json:"description,omitempty"`
	Type          string         `json:"type"`
	PrinterType   *string        `json:"printer_type,omitempty"`
	ServerAddress *string        `json:"server_address,omitempty"`
	VendorID      *string        `json:"vendor_id,omitempty"`
	ProductID     *string        `json:"product_id,omitempty"`
	TenantID      *string        `json:"tenant_id,omitempty"`
	AuditMetadata *AuditResponse `json:"audit_metadata,omitempty"`
}

type SyslogProfileDTO struct {
	ID             *string `json:"id,omitempty"`
	ProfileID      *string `json:"profile_id,omitempty"`
	ProfileName    string  `json:"profile_name"`
	Address        string  `json:"address"`
	Port           int64   `json:"port"`
	AlertEmail     string  `json:"alert_email"`
	Format         string  `json:"format"`
	ConnectionType *string `json:"connection_type,omitempty"`
	Cert           *string `json:"cert,omitempty"`
	Status         *string `json:"status,omitempty"`
}

type ForwardProfileFilterDTO struct {
	Region []string `json:"region,omitempty"`
}

type ForwardProfileDTO struct {
	ID        *string                  `json:"id,omitempty"`
	ProfileID *string                  `json:"profile_id,omitempty"`
	Name      string                   `json:"name"`
	Enabled   bool                     `json:"enabled"`
	Filter    *ForwardProfileFilterDTO `json:"filter,omitempty"`
}
