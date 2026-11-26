# List DLP data patterns, optionally filtered by name.
data "scm_dlp_data_pattern_list" "all" {}

data "scm_dlp_data_pattern_list" "by_name" {
  name = "REPLACE_WITH_NAME"
}
