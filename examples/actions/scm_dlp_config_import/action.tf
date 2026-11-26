# Import a DLP configuration bundle previously produced by scm_dlp_config_export,
# overwriting conflicting objects.
#
# Usage:
#   terraform apply -invoke=action.scm_dlp_config_import.restore

action "scm_dlp_config_import" "restore" {
  config {
    file_path       = "${path.module}/dlp-config-export.json"
    conflict_action = "OVERWRITE"
  }
}
