terraform {
  required_providers {
    scm = {
      source  = "paloaltonetworks-local/scm"
      version = "1.0.0"
    }
  }
}

# Uses DLP_API_URL, DLP_AUTH_URL, DLP_CLIENT_ID, DLP_CLIENT_SECRET, DLP_TSG_ID from environment
provider "scm" {}

# 1. Read tenant channels from live tenant
data "scm_dlp_tenant_channels" "current" {}

# 2. Read OCR enablement for prisma-access from live tenant
data "scm_dlp_ocr_enablement" "prisma_access" {
  target_service_name = "prisma-access"
}

# 3. Read tenant-wide DLP configuration (snippet settings, EDM status, etc.)
data "scm_dlp_tenant_config" "current" {}

# 4. Look up predefined Credit Card pattern by name on live tenant
data "scm_dlp_data_pattern_list" "ccn" {
  name = "Credit Card Numbers"
}

# 5. Create a temporary custom weighted-regex pattern for verification
resource "scm_dlp_data_pattern" "tf_live_verify" {
  name        = "TF-Live-Verify-EmpID-Pattern"
  description = "Created by Terraform provider live verification test"
  type        = "custom"

  detection_config = {
    technique                   = "weighted_regex"
    supported_confidence_levels = ["low", "medium", "high"]
  }

  matching_rules = {
    proximity_distance = 50
    regexes = [
      {
        regex  = "TFEMP-[0-9]{6}"
        weight = 10
      }
    ]
  }
}

# 6. Create a temporary Data Profile referencing the custom pattern
resource "scm_dlp_data_profile" "tf_live_verify" {
  name                     = "TF-Live-Verify-Profile"
  description              = "Created by Terraform provider live verification test"
  is_granular_data_profile = false

  detection_rules = [
    {
      rule_type = "expression_tree"
      expression_tree = {
        operator_type = "or"
        sub_expressions = [
          {
            rule_item = {
              id                       = scm_dlp_data_pattern.tf_live_verify.id
              name                     = scm_dlp_data_pattern.tf_live_verify.name
              detection_technique      = "weighted_regex"
              occurrence_operator_type = "any"
              confidence_level         = "high"
              match_type               = "include"
              by_unique_count          = false
            }
          }
        ]
      }
    }
  ]
}

output "tenant_channels" {
  value = data.scm_dlp_tenant_channels.current.channels
}

output "ocr_prisma_access" {
  value = data.scm_dlp_ocr_enablement.prisma_access.ocr_enablement
}

output "tenant_edm_enablement" {
  value = data.scm_dlp_tenant_config.current.edm_enablement
}

output "created_pattern_id" {
  value = scm_dlp_data_pattern.tf_live_verify.id
}

output "created_profile_id" {
  value = scm_dlp_data_profile.tf_live_verify.id
}

data "scm_dlp_dry_run_scan" "verify_emp_match" {
  file_name             = "sample-emp.txt"
  file_content          = "Confidential Employee Record: TFEMP-123456 and TFEMP-654321"
  source_data_profile_id = scm_dlp_data_profile.tf_live_verify.id
  data_profile = jsonencode({
    name                     = scm_dlp_data_profile.tf_live_verify.name
    profile_type             = scm_dlp_data_profile.tf_live_verify.profile_type
    profile_status           = scm_dlp_data_profile.tf_live_verify.profile_status
    is_granular_data_profile = false
    detection_rules = [
      {
        rule_type       = "expression_tree"
        expression_tree = jsondecode(scm_dlp_data_profile.tf_live_verify.detection_rules[0].expression_tree.expression_tree_json)
      }
    ]
  })
}

output "dry_run_status" {
  value = data.scm_dlp_dry_run_scan.verify_emp_match.status
}

output "dry_run_result" {
  value = jsondecode(data.scm_dlp_dry_run_scan.verify_emp_match.result_json)
}
