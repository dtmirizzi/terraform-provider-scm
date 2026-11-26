# Every data profile gets a companion data filtering profile from the backend.
# This resource configures it; destroying it only removes it from state.
resource "scm_dlp_data_filtering_profile" "pci_enforcement" {
  data_profile_id = tonumber(scm_dlp_data_profile.pci_and_employee_pii.id)
  direction       = "both"
  scan_type       = "include"
  file_based      = true
  non_file_based  = true
  file_type       = ["pdf", "csv", "docx", "xlsx"]
  log_severity    = "HIGH"
  rule1_action    = "block"

  app_exclusion_list = [
    {
      app_id   = "jira"
      app_name = "Jira"
      type     = "application-filter"
    },
  ]
}
