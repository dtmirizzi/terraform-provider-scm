terraform {
  required_providers {
    scm = {
      source  = "paloaltonetworks-local/scm"
      version = "1.0.0"
    }
  }
}

provider "scm" {
  dlp_host         = "api.dlp.paloaltonetworks.com"
  dlp_service_name = "prisma-access"
}

# 1. Tenant-wide DLP configuration: enable snippets, partial masking, alert exemptions, and EDM
resource "scm_dlp_tenant_config" "global" {
  enable_snippets                           = true
  mask_level                                = "PARTIAL_MASK"
  incident_notification_enablement          = true
  incident_notification_snippets_enablement = true
  detection_exemption_period_minutes        = 60
  enable_structured_data_clustering         = true
  edm_enablement                            = "REQUESTED"
}

# 2. OCR enablement for Prisma Access traffic
resource "scm_dlp_ocr_enablement" "prisma_access" {
  target_service_name = "prisma-access"
  ocr_enablement      = "ENABLED"
}

# 3. Alert & notification exemption settings
resource "scm_dlp_alert_settings" "default" {
  incident_notification_enablement   = true
  detection_exemption_period_minutes = 60
}

# 4. Query licensed DLP channels for the tenant
data "scm_dlp_tenant_channels" "active" {}

# 5. Look up predefined Credit Card pattern by name
data "scm_dlp_data_pattern_list" "credit_card" {
  name = "Credit Card Numbers"
}

# 6. Look up an EDM dataset by name
data "scm_dlp_edm_dataset_list" "customer_pii" {
  name = "Customer_PII_Dataset"
}

# 7. Create a custom keyword dictionary (auto-uploads keywords via multipart PUT)
resource "scm_dlp_dictionary" "project_codenames" {
  name               = "Internal-Project-Codenames"
  description        = "Confidential project codenames"
  category           = "Confidential"
  region_name        = "us"
  original_file_name = "codenames.txt"
  is_case_sensitive  = false
  keyword_list       = ["ProjectApollo", "ProjectOrion", "ProjectCentaurus"]
}

# 8. Create a custom weighted-regex data pattern
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
    regexes = [
      {
        regex  = "EMP-[0-9]{6}"
        weight = 10
      }
    ]
  }
}

# 9. Create a Data Profile combining Predefined Regex, Custom Pattern, and Dictionary
resource "scm_dlp_data_profile" "pci_and_employee_pii" {
  name                     = "PCI-and-Employee-PII"
  description              = "Triggers on 5+ unique CCNs or Employee ID + Project Codename"
  is_granular_data_profile = false

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
              match_type               = "include"
              by_unique_count          = true
            }
          },
          {
            operator_type = "and"
            sub_expressions = [
              {
                rule_item = {
                  id                       = scm_dlp_data_pattern.employee_id.id
                  detection_technique      = "weighted_regex"
                  occurrence_operator_type = "any"
                  confidence_level         = "high"
                  match_type               = "include"
                }
              },
              {
                rule_item = {
                  id                       = scm_dlp_dictionary.project_codenames.id
                  detection_technique      = "dictionary"
                  occurrence_operator_type = "any"
                  confidence_level         = "high"
                  match_type               = "include"
                }
              }
            ]
          }
        ]
      }
    }
  ]
}

# 10. Configure the linked Data Filtering Profile (inclusions/exclusions/actions)
resource "scm_dlp_data_filtering_profile" "pci_enforcement" {
  data_profile_id = tonumber(scm_dlp_data_profile.pci_and_employee_pii.id)
  direction       = "both"
  scan_type       = "include"
  file_based      = true
  non_file_based  = true
  file_type       = ["pdf", "csv", "docx", "xlsx"]
  log_severity    = "HIGH"
  rule1_action    = "block"

  app_exclusion_list = [
    {
      app_id   = "jira"
      app_name = "Jira"
      type     = "application-filter"
    }
  ]
}
