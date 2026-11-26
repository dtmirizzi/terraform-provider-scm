# Tenant-wide Enterprise DLP settings (singleton). Destroying this resource only
# removes it from state; the tenant keeps its current settings.
#
# Incident notifications and the detection exemption period are managed by
# scm_dlp_alert_settings, and OCR by scm_dlp_ocr_enablement.
resource "scm_dlp_tenant_config" "global" {
  enable_snippets                           = true
  mask_level                                = "PARTIAL_MASK"
  incident_notification_snippets_enablement = true
  enable_structured_data_clustering         = true
  edm_enablement                            = "REQUESTED"
}
