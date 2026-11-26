# Forward DLP incident logs from selected regions.
resource "scm_dlp_forward_profile" "us_eu" {
  name    = "US-EU-Forwarding"
  enabled = true
  regions = ["us", "eu"]
}
