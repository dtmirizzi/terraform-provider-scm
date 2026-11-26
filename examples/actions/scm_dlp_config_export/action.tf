# Export the tenant's DLP configuration.
#
# Usage:
#   terraform apply -invoke=action.scm_dlp_config_export.backup

action "scm_dlp_config_export" "backup" {
  config {
    wait_for_completion = true
    timeout_ms          = 120000
  }
}
