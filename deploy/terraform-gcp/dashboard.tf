resource "google_compute_instance" "dashboard" {
  name         = "logistics-dashboard"
  tags         = ["logistics-dashboard", "singlestore-logistics-sim"]
  machine_type = var.dashboard_machine_type

  boot_disk {
    initialize_params {
      image = var.machine_image
    }
  }

  scratch_disk {
    // 375 GB local SSD drive.
    interface = "NVME"
  }

  network_interface {
    network = google_compute_network.vpc_network.self_link
    access_config {}
  }

  service_account {
    email  = google_service_account.service_account.email
    scopes = local.default_scopes
  }

  metadata = {
    startup-script-url = "gs://${google_storage_bucket.default.name}/${google_storage_bucket_object.setup_dashboard.output_name}"
    node-exporter-targets = jsonencode(concat(
      [for i, _ in google_compute_instance.simulator[*] : "simulator-${i}:9100"],
      [for i, _ in google_compute_instance.singlestore_agg[*] : "s2-agg-${i}:9100"],
      [for i, _ in google_compute_instance.singlestore_leaf[*] : "s2-leaf-${i}:9100"],
      [for i, _ in google_compute_instance.redpanda[*] : "rp-node-${i}:9100"],
      ["logistics-dashboard:9100"],
    )),
    simulator-targets = jsonencode([for i, _ in google_compute_instance.simulator[*] : "simulator-${i}:9000"])
    redpanda-targets  = jsonencode([for i, _ in google_compute_instance.redpanda[*] : "rp-node-${i}:9644"])
    dashboards        = join(" ", [for k, v in google_storage_bucket_object.grafana_dashboards : "gs://${google_storage_bucket.default.name}/${v.output_name}"])
  }
}
