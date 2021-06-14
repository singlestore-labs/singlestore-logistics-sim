locals {
  script_prelude = [
    file("${path.module}/scripts/lib.sh"),
    file("${path.module}/scripts/setup-disks.sh"),
    file("${path.module}/scripts/setup-apt.sh"),
    file("${path.module}/scripts/setup-node-exporter.sh"),
  ]
}

resource "google_storage_bucket" "default" {
  name                        = var.storage_bucket
  uniform_bucket_level_access = true
}

resource "google_storage_bucket_object" "grafana_dashboards" {
  for_each = fileset("${path.module}/../../data/metrics/dashboards", "*.json")

  name   = "dashboards/${each.value}"
  bucket = google_storage_bucket.default.name
  source = "${path.module}/../../data/metrics/dashboards/${each.value}"
}

resource "null_resource" "build_simulator" {
  provisioner "local-exec" {
    working_dir = "${path.module}/../../simulator"
    command     = "DOCKER_BUILDKIT=1 docker build --target bin --output bin/simulator ."
  }
}

resource "google_storage_bucket_object" "simulator" {
  depends_on = [
    null_resource.build_simulator
  ]

  name   = "bin/simulator"
  bucket = google_storage_bucket.default.name
  source = "${path.module}/../../simulator/bin/simulator/simulator"
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

resource "google_storage_bucket_object" "schema" {
  name   = "data/schema.sql"
  bucket = google_storage_bucket.default.name
  source = "${path.module}/../../schema.sql"
}

resource "google_storage_bucket_object" "worldcities" {
  name   = "data/worldcities.csv"
  bucket = google_storage_bucket.default.name
  source = "${path.module}/../../data/simplemaps/worldcities.csv"
}
