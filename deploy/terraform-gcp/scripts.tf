locals {
  script_prelude = [
    file("${path.module}/scripts/lib.sh"),
    file("${path.module}/scripts/setup-disks.sh"),
    file("${path.module}/scripts/setup-apt.sh"),
  ]
}

resource "google_storage_bucket" "default" {
  name                        = var.storage_bucket
  uniform_bucket_level_access = true
}

resource "google_storage_bucket_iam_member" "service_account" {
  bucket = google_storage_bucket.default.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.service_account.email}"
}

resource "google_storage_bucket_object" "grafana_dashboards" {
  for_each = fileset("${path.module}/../../data/metrics/dashboards", "*.json")

  name   = "dashboards/${each.value}"
  bucket = google_storage_bucket.default.name
  source = "${path.module}/../../data/metrics/dashboards/${each.value}"
}

resource "google_storage_bucket_object" "simulator" {
  name   = "bin/simulator"
  bucket = google_storage_bucket.default.name
  source = "${path.module}/../../simulator/bin/simulator/simulator"

  provisioner "local-exec" {
    working_dir = "${path.module}/../../simulator/bin/simulator"
    command     = "go build"
  }
}

resource "google_storage_bucket_object" "setup_dashboard" {
  name   = "scripts/setup_dashboard.sh"
  bucket = google_storage_bucket.default.name
  content = join("\n", concat(local.script_prelude, [
    file("${path.module}/scripts/setup-docker.sh"),
    file("${path.module}/scripts/setup-prometheus.sh"),
    file("${path.module}/scripts/setup-grafana.sh"),
    file("${path.module}/scripts/setup-singlestore-studio.sh"),
  ]))
}

resource "google_storage_bucket_object" "setup_redpanda" {
  name   = "scripts/setup_redpanda.sh"
  bucket = google_storage_bucket.default.name
  content = join("\n", concat(local.script_prelude, [
    file("${path.module}/scripts/setup-redpanda.sh"),
  ]))
}

resource "google_storage_bucket_object" "setup_singlestore_agg" {
  name   = "scripts/setup_singlestore_agg.sh"
  bucket = google_storage_bucket.default.name
  content = join("\n", concat(local.script_prelude, [
    file("${path.module}/scripts/setup-singlestore-base.sh"),
    file("${path.module}/scripts/setup-singlestore-agg.sh"),
  ]))
}

resource "google_storage_bucket_object" "setup_singlestore_leaf" {
  name   = "scripts/setup_singlestore_leaf.sh"
  bucket = google_storage_bucket.default.name
  content = join("\n", concat(local.script_prelude, [
    file("${path.module}/scripts/setup-singlestore-base.sh"),
  ]))
}

resource "google_storage_bucket_object" "setup_simulator" {
  name   = "scripts/setup_simulator.sh"
  bucket = google_storage_bucket.default.name
  content = join("\n", concat(local.script_prelude, [
    file("${path.module}/scripts/setup-simulator.sh"),
  ]))
}
