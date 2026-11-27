# [Beta / Experimental] Invoke a positive and negative dry-run test suite against a data profile.
#
# Usage:
#   terraform apply -invoke=action.scm_dlp_dry_run_scan.check

action "scm_dlp_dry_run_scan" "check" {
  config {
    data_profile_id  = scm_dlp_data_profile.employee_pii.id
    fail_on_mismatch = true

    test_cases = [
      {
        name             = "positive-sample-file"
        file_path        = "${path.module}/testdata/positive/sample_sensitive.txt"
        expected_verdict = "MATCHED"
        expected_action  = "block"
      },
      {
        name             = "negative-benign-file"
        file_path        = "${path.module}/testdata/negative/sample_benign.txt"
        expected_verdict = "NOT_MATCHED"
      }
    ]
  }
}
