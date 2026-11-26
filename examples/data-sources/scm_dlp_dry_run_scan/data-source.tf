# Scan sample content against a candidate data profile without enforcing it.
# Note: as a data source this uploads the content on every plan/refresh; prefer
# the scm_dlp_dry_run_scan action for one-off scans.
data "scm_dlp_dry_run_scan" "sample" {
  file_content = "Employee EMP-123456 card 4111 1111 1111 1111"
  file_name    = "sample.txt"
  data_profile = jsonencode({
    name = "dry-run"
    detection_rules = [{
      rule_type = "expression_tree"
      expression_tree = {
        operator_type = "or"
        sub_expressions = [{
          rule_item = {
            id                       = "REPLACE_WITH_DATA_PATTERN_ID"
            detection_technique      = "weighted_regex"
            occurrence_operator_type = "any"
          }
        }]
      }
    }]
  })
}
