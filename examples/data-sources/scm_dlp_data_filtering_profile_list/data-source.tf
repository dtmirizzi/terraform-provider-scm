# List DLP data filtering profiles, optionally filtered by name.
data "scm_dlp_data_filtering_profile_list" "all" {}

data "scm_dlp_data_filtering_profile_list" "by_name" {
  name = "REPLACE_WITH_NAME"
}
