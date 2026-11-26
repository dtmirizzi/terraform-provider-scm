# Incident notification and alert exemption settings (singleton).
resource "scm_dlp_alert_settings" "default" {
  incident_notification_enablement   = true
  detection_exemption_period_minutes = 60
}
