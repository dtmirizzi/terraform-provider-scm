# Read the status of a config import job.
data "scm_dlp_config_import_job" "latest" {
  import_job_id = "REPLACE_WITH_JOB_ID"
}
