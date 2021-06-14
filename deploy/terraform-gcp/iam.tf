resource "google_service_account" "service_account" {
  account_id   = "singlestore-logistics-sim"
  display_name = "SingleStore Logistics Sim"
}

locals {
  // rather than managing scopes per instance, we manage permissions via
  // assigning iam roles to the service account
  default_scopes = ["https://www.googleapis.com/auth/cloud-platform"]
}

resource "google_project_iam_member" "logging" {
  project = var.project_name
  member  = "serviceAccount:${google_service_account.service_account.email}"
  role    = "roles/logging.logWriter"
}

resource "google_project_iam_member" "monitoring" {
  project = var.project_name
  member  = "serviceAccount:${google_service_account.service_account.email}"
  role    = "roles/monitoring.metricWriter"
}

resource "google_project_iam_member" "tracing" {
  project = var.project_name
  member  = "serviceAccount:${google_service_account.service_account.email}"
  role    = "roles/cloudtrace.agent"
}

resource "google_storage_bucket_iam_member" "service_account" {
  bucket = google_storage_bucket.default.name
  member = "serviceAccount:${google_service_account.service_account.email}"
  role   = "roles/storage.objectAdmin"
}
