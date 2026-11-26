# Adopt an EDM dataset that was already uploaded and indexed with the EDM CLI.
# Terraform never creates or deletes EDM data: destroying this resource only
# removes it from state.
resource "scm_dlp_edm_dataset" "customer_pii" {
  name = "Customer_PII_Dataset"
}
