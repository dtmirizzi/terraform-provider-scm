# [Beta / Experimental] Validate a Data Profile against both positive and negative test cases.
#
# You can reference an existing `scm_dlp_data_profile` via `data_profile_id` (or pass raw
# JSON via `data_profile`) and supply a list of `test_cases` (files or inline content)
# with `expected_verdict` ("MATCHED" for positive tests, "NOT_MATCHED" for negative tests)
# and optional `expected_action`. When `fail_on_mismatch = true` (default), Terraform fails
# the plan/apply if any positive test fails to match or any negative test produces a false positive.

data "scm_dlp_dry_run_scan" "profile_regression_suite" {
  data_profile_id  = scm_dlp_data_profile.employee_pii.id
  fail_on_mismatch = true

  test_cases = [
    {
      name             = "positive-employee-id-export"
      file_path        = "${path.module}/testdata/positive/employee_records.csv"
      expected_verdict = "MATCHED"
      expected_action  = "block"
    },
    {
      name             = "positive-inline-card-sample"
      file_content     = "Employee EMP-123456 corporate card 4111 1111 1111 1111"
      file_name        = "inline_positive.txt"
      expected_verdict = "MATCHED"
      expected_action  = "block"
    },
    {
      name             = "negative-benign-order-numbers"
      file_path        = "${path.module}/testdata/negative/order_numbers.txt"
      expected_verdict = "NOT_MATCHED"
    },
    {
      name             = "negative-public-release-notes"
      file_content     = "General product release notes with ticket reference 987654."
      file_name        = "release_notes.md"
      expected_verdict = "NOT_MATCHED"
    }
  ]
}
