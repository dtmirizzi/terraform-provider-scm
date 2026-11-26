# An approved USB storage device, identified by vendor and product ID.
resource "scm_dlp_peripheral" "approved_usb" {
  name       = "Approved-SanDisk-Cruzer"
  type       = "usb"
  vendor_id  = "0781"
  product_id = "5581"
}
