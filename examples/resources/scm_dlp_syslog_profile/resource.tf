# Forward DLP incidents to a syslog collector.
resource "scm_dlp_syslog_profile" "siem" {
  profile_name    = "SIEM-Collector"
  address         = "10.0.0.10"
  port            = 514
  connection_type = "TCP"
  format          = "CEF"
  alert_email     = "secops@example.com"
}
