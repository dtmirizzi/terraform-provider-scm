# A document type used for document fingerprinting.
resource "scm_dlp_document_type" "contracts" {
  name                = "Customer-Contracts"
  description         = "Fingerprint of the standard customer contract template"
  category            = "Legal"
  detection_technique = "document_fingerprint"
  original_file_name  = "customer-contract.pdf"
}
