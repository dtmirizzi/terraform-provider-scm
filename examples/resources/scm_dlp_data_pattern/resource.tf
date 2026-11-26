# A custom weighted-regex data pattern for internal employee IDs.
resource "scm_dlp_data_pattern" "employee_id" {
  name        = "Custom-Employee-ID"
  description = "Detects internal employee ID format EMP-XXXXXX"
  type        = "custom"

  detection_config = {
    technique                   = "weighted_regex"
    supported_confidence_levels = ["low", "medium", "high"]
  }

  matching_rules = {
    proximity_distance = 50
    proximity_keywords = ["employee", "emp id"]
    regexes = [
      {
        regex  = "EMP-[0-9]{6}"
        weight = 10
      },
    ]
  }
}
