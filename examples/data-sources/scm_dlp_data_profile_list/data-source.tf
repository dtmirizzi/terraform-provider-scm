# List DLP data profiles, optionally filtered by name.
data "scm_dlp_data_profile_list" "all" {}

data "scm_dlp_data_profile_list" "by_name" {
  name = "REPLACE_WITH_NAME"
}
