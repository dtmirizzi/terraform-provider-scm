# A custom keyword dictionary. Keywords are uploaded as a file on create and
# update. If keyword_list is omitted on update, the existing keywords are kept.
resource "scm_dlp_dictionary" "project_codenames" {
  name               = "Internal-Project-Codenames"
  description        = "Confidential project codenames"
  category           = "Confidential"
  region_name        = "us"
  original_file_name = "codenames.txt"
  is_case_sensitive  = false
  keyword_list       = ["ProjectApollo", "ProjectOrion", "ProjectCentaurus"]
}
