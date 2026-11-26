package dlpclient

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	DetectionTechniques = []string{
		"regex",
		"weighted_regex",
		"ml",
		"edm",
		"dictionary",
		"document_fingerprint",
		"trainable_classifier",
		"ml_document",
		"titus_tag",
		"wildfire",
		"file_property",
		"pab",
		"document_classifier",
	}
	OccurrenceOperators = []string{"any", "less_than_equal_to", "more_than_equal_to", "between"}
	ConfidenceLevels    = []string{"low", "medium", "high"}
	MatchTypes          = []string{"include", "exclude"}
	RuleOperatorTypes   = []string{"and", "or", "not", "and_not", "or_not"}
	DetectionRuleTypes  = []string{"expression_tree", "multi_profile"}
	ResourceCategories  = []string{
		"Academic",
		"Confidential",
		"Employment",
		"Financial",
		"Government",
		"Healthcare",
		"Legal",
		"Marketing",
		"Source Code",
	}
	SeverityLevels     = []string{"informational", "low", "medium", "high", "critical"}
	LogSeverityLevels  = []string{"CRITICAL", "HIGH", "MEDIUM", "LOW", "INFORMATIONAL"}
	PatternTypes       = []string{"predefined", "custom", "file_property"}
	DocTypeTechniques  = []string{"document_fingerprint", "trainable_classifier"}
	EndpointPolicyType = []string{"endpoint_peripheral_control", "endpoint_data_in_motion", "network"}
	PeripheralTypes    = []string{"usb", "printer", "network_share"}
	OcrStates          = []string{"REQUESTED", "ENABLED", "DISABLED"}
	OcrServices        = []string{
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
	}
	SyslogFormats   = []string{"IETF", "CSV", "LEEF", "CEF", "LEEF 1.0"}
	SyslogConnTypes = []string{"TCP", "UDP"}
	ForwardRegions  = []string{"us", "eu", "ap", "uk", "in", "au", "jp", "ca", "par", "sui", "br", "sau"}
	MaskLevels      = []string{"NO_MASK", "PARTIAL_MASK", "FULL_MASK"}
	EdmStates       = []string{"REQUESTED", "ENABLED", "DISABLED"}
)

const (
	MaxDictionaryKeywords = 2500
	MaxKeywordLength      = 128
)

// NormalizeAndValidateDataProfile validates a DataProfileDTO and fills backend defaults
// (match_type = "include", confidence_level = "high", version = 1, by_unique_count = false).
func NormalizeAndValidateDataProfile(profile *DataProfileDTO) ([]string, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is nil")
	}
	var errs []string

	if strings.TrimSpace(profile.Name) == "" {
		errs = append(errs, "'name' is required and must not be blank")
	} else if len(profile.Name) > 64 {
		errs = append(errs, fmt.Sprintf("'name' must be at most 64 characters (got %d)", len(profile.Name)))
	}

	if len(profile.DetectionRules) == 0 {
		errs = append(errs, "'detection_rules' is required and must be a non-empty array")
		return errs, fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}

	hasExpr := false
	hasMulti := false

	for i := range profile.DetectionRules {
		rule := &profile.DetectionRules[i]
		prefix := fmt.Sprintf("detection_rules[%d]", i)
		if !slices.Contains(DetectionRuleTypes, rule.RuleType) {
			errs = append(errs, fmt.Sprintf("%s.rule_type must be one of: %s (got: %q)", prefix, strings.Join(DetectionRuleTypes, ", "), rule.RuleType))
			continue
		}
		switch rule.RuleType {
		case "expression_tree":
			hasExpr = true
			if rule.ExpressionTree == nil {
				errs = append(errs, fmt.Sprintf("%s.expression_tree is required when rule_type is 'expression_tree'", prefix))
			} else {
				validateAndNormalizeExpressionTree(rule.ExpressionTree, prefix+".expression_tree", &errs)
			}
		case "multi_profile":
			hasMulti = true
			if rule.MultiProfile == nil {
				errs = append(errs, fmt.Sprintf("%s.multi_profile is required when rule_type is 'multi_profile'", prefix))
			} else {
				if rule.MultiProfile.OperatorType != "or" {
					errs = append(errs, fmt.Sprintf("%s.multi_profile.operator_type must be 'or' (got: %q)", prefix, rule.MultiProfile.OperatorType))
				}
				if len(rule.MultiProfile.DataProfileIDs) == 0 {
					errs = append(errs, fmt.Sprintf("%s.multi_profile.data_profile_ids is required and must be non-empty", prefix))
				}
			}
		}
	}

	if hasExpr && hasMulti {
		errs = append(errs, "Mixed rule types are not allowed: detection_rules must be either all 'expression_tree' or all 'multi_profile'")
	}

	if profile.IsGranularDataProfile != nil && *profile.IsGranularDataProfile {
		if hasExpr {
			errs = append(errs, "Granular data profile must use 'multi_profile' as the detection rule type")
		}
		if len(profile.DetectionRules) != 1 {
			errs = append(errs, fmt.Sprintf("Granular data profile must have exactly one detection rule (got %d)", len(profile.DetectionRules)))
		}
	}

	if profile.ProfileType == nil || *profile.ProfileType == "" {
		v := "advanced"
		profile.ProfileType = &v
	}
	if profile.ProfileStatus == nil || *profile.ProfileStatus == "" {
		v := "active"
		profile.ProfileStatus = &v
	}

	if len(errs) > 0 {
		return errs, fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}
	return nil, nil
}

func validateAndNormalizeExpressionTree(node *ExpressionTreeNodeDTO, path string, errs *[]string) {
	if node.RuleItem != nil {
		validateAndNormalizeRuleItem(node.RuleItem, path+".rule_item", errs)
		if node.SubExpressions == nil {
			node.SubExpressions = []ExpressionTreeNodeDTO{}
		}
		return
	}

	if node.OperatorType != nil && *node.OperatorType != "" {
		if !slices.Contains(RuleOperatorTypes, *node.OperatorType) {
			*errs = append(*errs, fmt.Sprintf("%s.operator_type must be one of: %s (got: %q)", path, strings.Join(RuleOperatorTypes, ", "), *node.OperatorType))
		}
		if len(node.SubExpressions) == 0 {
			*errs = append(*errs, fmt.Sprintf("%s.sub_expressions is required and must be non-empty when operator_type is set", path))
			return
		}
		for i := range node.SubExpressions {
			validateAndNormalizeExpressionTree(&node.SubExpressions[i], fmt.Sprintf("%s.sub_expressions[%d]", path, i), errs)
		}
		return
	}

	*errs = append(*errs, fmt.Sprintf("%s: invalid node structure: must have either 'rule_item' or 'operator_type' + non-empty 'sub_expressions'", path))
}

func validateAndNormalizeRuleItem(item *RuleItemDTO, path string, errs *[]string) {
	if strings.TrimSpace(item.ID) == "" {
		*errs = append(*errs, fmt.Sprintf("%s.id is required", path))
	}
	if !slices.Contains(DetectionTechniques, item.DetectionTechnique) {
		*errs = append(*errs, fmt.Sprintf("%s.detection_technique must be one of: %s (got: %q)", path, strings.Join(DetectionTechniques, ", "), item.DetectionTechnique))
	}
	if !slices.Contains(OccurrenceOperators, item.OccurrenceOperatorType) {
		*errs = append(*errs, fmt.Sprintf("%s.occurrence_operator_type must be one of: %s (got: %q)", path, strings.Join(OccurrenceOperators, ", "), item.OccurrenceOperatorType))
	} else {
		switch item.OccurrenceOperatorType {
		case "more_than_equal_to", "less_than_equal_to":
			if item.OccurrenceCount == nil {
				*errs = append(*errs, fmt.Sprintf("%s.occurrence_count is required when occurrence_operator_type is '%s'", path, item.OccurrenceOperatorType))
			}
		case "between":
			if item.OccurrenceLow == nil {
				*errs = append(*errs, fmt.Sprintf("%s.occurrence_low is required when occurrence_operator_type is 'between'", path))
			}
			if item.OccurrenceHigh == nil {
				*errs = append(*errs, fmt.Sprintf("%s.occurrence_high is required when occurrence_operator_type is 'between'", path))
			}
		}
	}

	if item.ConfidenceLevel != nil && *item.ConfidenceLevel != "" {
		if !slices.Contains(ConfidenceLevels, *item.ConfidenceLevel) {
			*errs = append(*errs, fmt.Sprintf("%s.confidence_level must be one of: %s", path, strings.Join(ConfidenceLevels, ", ")))
		}
	} else {
		v := "high"
		item.ConfidenceLevel = &v
	}

	if item.MatchType != nil && *item.MatchType != "" {
		if !slices.Contains(MatchTypes, *item.MatchType) {
			*errs = append(*errs, fmt.Sprintf("%s.match_type must be one of: %s", path, strings.Join(MatchTypes, ", ")))
		}
	} else {
		v := "include"
		item.MatchType = &v
	}

	if item.Version == nil {
		v := int64(1)
		item.Version = &v
	}
	if item.ByUniqueCount == nil {
		v := false
		item.ByUniqueCount = &v
	}
}

// ValidateDataPattern validates a DataPatternDTO before sending to the backend.
func ValidateDataPattern(pattern *DataPatternDTO) ([]string, error) {
	if pattern == nil {
		return nil, fmt.Errorf("pattern is nil")
	}
	var errs []string
	if strings.TrimSpace(pattern.Name) == "" {
		errs = append(errs, "'name' is required")
	} else if len(pattern.Name) > 64 {
		errs = append(errs, fmt.Sprintf("'name' must be at most 64 characters (got %d)", len(pattern.Name)))
	}
	if !slices.Contains(PatternTypes, pattern.Type) {
		errs = append(errs, fmt.Sprintf("'type' must be one of: %s (got: %q)", strings.Join(PatternTypes, ", "), pattern.Type))
	}
	if pattern.DetectionConfig == nil {
		errs = append(errs, "'detection_config' is required")
	} else {
		if !slices.Contains(DetectionTechniques, pattern.DetectionConfig.Technique) {
			errs = append(errs, fmt.Sprintf("'detection_config.technique' must be one of: %s (got: %q)", strings.Join(DetectionTechniques, ", "), pattern.DetectionConfig.Technique))
		}
		if len(pattern.DetectionConfig.SupportedConfidenceLevels) == 0 {
			errs = append(errs, "'detection_config.supported_confidence_levels' is required and must be non-empty")
		} else {
			for _, lvl := range pattern.DetectionConfig.SupportedConfidenceLevels {
				if !slices.Contains(ConfidenceLevels, lvl) {
					errs = append(errs, fmt.Sprintf("'detection_config.supported_confidence_levels' contains invalid value: %q", lvl))
				}
			}
		}
		if pattern.DetectionConfig.Technique == "weighted_regex" || pattern.DetectionConfig.Technique == "regex" {
			if pattern.MatchingRules == nil || len(pattern.MatchingRules.Regexes) == 0 {
				errs = append(errs, fmt.Sprintf("'matching_rules.regexes' is required when technique is '%s'", pattern.DetectionConfig.Technique))
			} else {
				for i, rx := range pattern.MatchingRules.Regexes {
					if strings.TrimSpace(rx.Regex) == "" {
						errs = append(errs, fmt.Sprintf("matching_rules.regexes[%d].regex is required", i))
					}
				}
			}
		}
		if pattern.MatchingRules != nil && pattern.MatchingRules.ProximityDistance != nil {
			pd := *pattern.MatchingRules.ProximityDistance
			if pd < 2 || pd > 1000 {
				errs = append(errs, "'matching_rules.proximity_distance' must be between 2 and 1000")
			}
		}
	}

	if len(errs) > 0 {
		return errs, fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}
	return nil, nil
}

func isShortKeywordScript(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul, unicode.Thai, unicode.Khmer) {
			return true
		}
	}
	return false
}

// BuildDictionaryUpdateParts validates a DictionaryDTO and builds the multipart PUT parts (json + file).
func BuildDictionaryUpdateParts(dict *DictionaryDTO, keywordList []string) ([]MultipartPart, []string, error) {
	if dict == nil {
		return nil, nil, fmt.Errorf("dictionary is nil")
	}
	var errs []string
	if strings.TrimSpace(dict.Name) == "" {
		errs = append(errs, "'name' is required")
	}
	if strings.TrimSpace(dict.OriginalFileName) == "" {
		errs = append(errs, "'original_file_name' is required")
	}
	if strings.TrimSpace(dict.RegionName) == "" {
		errs = append(errs, "'region_name' is required")
	}
	if !slices.Contains(ResourceCategories, dict.Category) {
		errs = append(errs, fmt.Sprintf("'category' must be one of: %s (got: %q)", strings.Join(ResourceCategories, ", "), dict.Category))
	}

	isCSV := strings.HasSuffix(dict.OriginalFileName, ".csv")
	if len(keywordList) > MaxDictionaryKeywords {
		errs = append(errs, fmt.Sprintf("keyword_list has more than %d entries", MaxDictionaryKeywords))
	}

	cleaned := make([]string, 0, len(keywordList))
	for i, raw := range keywordList {
		kw := strings.TrimSpace(raw)
		length := utf8.RuneCountInString(kw)
		minLen := 3
		if isShortKeywordScript(kw) {
			minLen = 1
		}
		switch {
		case strings.ContainsAny(kw, "\r\n"):
			errs = append(errs, fmt.Sprintf("keyword_list[%d] contains a line break", i))
		case isCSV && strings.Contains(kw, ","):
			errs = append(errs, fmt.Sprintf("keyword_list[%d] contains a comma, which CSV uploads reject", i))
		case length < minLen || length > MaxKeywordLength:
			errs = append(errs, fmt.Sprintf("keyword_list[%d] must be %d-%d characters after trimming (has %d)", i, minLen, MaxKeywordLength, length))
		}
		cleaned = append(cleaned, kw)
	}

	if len(errs) > 0 {
		return nil, errs, fmt.Errorf("dictionary validation failed: %s", strings.Join(errs, "; "))
	}

	metaBody := map[string]any{
		"name":               dict.Name,
		"category":           dict.Category,
		"original_file_name": dict.OriginalFileName,
		"region_name":        dict.RegionName,
	}
	if dict.Description != nil {
		metaBody["description"] = *dict.Description
	}
	if dict.IsCaseSensitive != nil {
		metaBody["is_case_sensitive"] = *dict.IsCaseSensitive
	}
	jsonBytes, err := json.Marshal(metaBody)
	if err != nil {
		return nil, nil, err
	}

	var fileContent strings.Builder
	fileContentType := "text/plain"
	fileName := "keywords.txt"
	if isCSV {
		fileContent.WriteString("keyword\n")
		fileContentType = "text/csv"
		fileName = "keywords.csv"
	}
	fileContent.WriteString(strings.Join(cleaned, "\n"))
	fileContent.WriteString("\n")

	parts := []MultipartPart{
		{
			FieldName:   "json",
			FileName:    "dictionary.json",
			ContentType: "application/json",
			Data:        jsonBytes,
		},
		{
			FieldName:   "file",
			FileName:    fileName,
			ContentType: fileContentType,
			Data:        []byte(fileContent.String()),
		},
	}

	return parts, nil, nil
}

// ValidateTenantCommonConfig validates tenant config fields (snippets, masking, exemption period, EDM).
func ValidateTenantCommonConfig(cfg *TenantCommonConfigDTO) ([]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("tenant config is nil")
	}
	var errs []string
	if cfg.MaskLevel != nil && *cfg.MaskLevel != "" && !slices.Contains(MaskLevels, *cfg.MaskLevel) {
		errs = append(errs, fmt.Sprintf("'mask_level' must be one of: %s (got: %q)", strings.Join(MaskLevels, ", "), *cfg.MaskLevel))
	}
	if cfg.OcrEnablement != nil && *cfg.OcrEnablement != "" && !slices.Contains(OcrStates, *cfg.OcrEnablement) {
		errs = append(errs, fmt.Sprintf("'ocr_enablement' must be one of: %s (got: %q)", strings.Join(OcrStates, ", "), *cfg.OcrEnablement))
	}
	if cfg.EdmEnablement != nil && *cfg.EdmEnablement != "" && !slices.Contains(EdmStates, *cfg.EdmEnablement) {
		errs = append(errs, fmt.Sprintf("'edm_enablement' must be one of: %s (got: %q)", strings.Join(EdmStates, ", "), *cfg.EdmEnablement))
	}
	if cfg.DetectionExemptionPeriodMinutes != nil {
		v := *cfg.DetectionExemptionPeriodMinutes
		if v < 1 || v > 129600 {
			errs = append(errs, "'detection_exemption_period_minutes' must be between 1 and 129600")
		}
	}
	if len(errs) > 0 {
		return errs, fmt.Errorf("tenant config validation failed: %s", strings.Join(errs, "; "))
	}
	return nil, nil
}
