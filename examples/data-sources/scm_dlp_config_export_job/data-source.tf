# Read the status (and download link, once completed) of a config export job.
data "scm_dlp_config_export_job" "latest" {
  export_job_id = "REPLACE_WITH_JOB_ID"
}
