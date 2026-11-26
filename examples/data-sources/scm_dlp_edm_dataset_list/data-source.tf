# List DLP edm datasets, optionally filtered by name.
data "scm_dlp_edm_dataset_list" "all" {}

data "scm_dlp_edm_dataset_list" "by_name" {
  name = "REPLACE_WITH_NAME"
}
