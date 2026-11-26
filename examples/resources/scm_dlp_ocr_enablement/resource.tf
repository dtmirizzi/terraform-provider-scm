# Enable OCR for Prisma Access traffic. Destroying this resource disables OCR
# for the target service.
resource "scm_dlp_ocr_enablement" "prisma_access" {
  target_service_name = "prisma-access"
  ocr_enablement      = "ENABLED"
}
