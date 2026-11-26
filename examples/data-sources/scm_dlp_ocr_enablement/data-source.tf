# Read the OCR enablement state for a service.
data "scm_dlp_ocr_enablement" "prisma_access" {
  target_service_name = "prisma-access"
}
