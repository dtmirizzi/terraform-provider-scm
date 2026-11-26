# Scan a local file against a candidate data profile without enforcing it.
#
# Usage:
#   terraform apply -invoke=action.scm_dlp_dry_run_scan.check

action "scm_dlp_dry_run_scan" "check" {
  config {
    file_path = "${path.module}/sample.txt"
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
}
