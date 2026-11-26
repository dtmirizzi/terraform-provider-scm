# An endpoint peripheral-control policy. scope_json and response_json are
# compared semantically, so formatting differences never cause drift.
resource "scm_dlp_endpoint_policy" "block_usb" {
  name        = "Block-Unapproved-USB"
  description = "Block copying to unapproved USB storage"
  type        = "endpoint_peripheral_control"
  severity    = "high"
  enabled     = true

  scope_json = jsonencode({
    user_groups = ["engineering"]
  })

  response_json = jsonencode({
    action = "block"
  })
}
