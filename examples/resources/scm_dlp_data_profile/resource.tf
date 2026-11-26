data "scm_dlp_data_pattern_list" "credit_card" {
  name = "Credit Card Numbers"
}

resource "scm_dlp_data_pattern" "employee_id" {
  name = "Custom-Employee-ID"
  type = "custom"

  detection_config = {
    technique                   = "weighted_regex"
    supported_confidence_levels = ["high"]
  }

  matching_rules = {
    regexes = [{ regex = "EMP-[0-9]{6}", weight = 10 }]
  }
}

# Triggers on 5+ unique credit card numbers, or on any employee ID.
resource "scm_dlp_data_profile" "pci_and_employee_pii" {
  name        = "PCI-and-Employee-PII"
  description = "5+ unique credit card numbers, or any employee ID"

  detection_rules = [
    {
      rule_type = "expression_tree"
      expression_tree = {
        operator_type = "or"
        sub_expressions = [
          {
            rule_item = {
              id                       = data.scm_dlp_data_pattern_list.credit_card.data[0].id
              name                     = "Credit Card Numbers"
              detection_technique      = "regex"
              occurrence_operator_type = "more_than_equal_to"
              occurrence_count         = 5
              confidence_level         = "high"
              by_unique_count          = true
            }
          },
          {
            rule_item = {
              id                       = scm_dlp_data_pattern.employee_id.id
              detection_technique      = "weighted_regex"
              occurrence_operator_type = "any"
            }
          },
        ]
      }
    },
  ]
}
