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
    scopes = ["default"]
  }
}
