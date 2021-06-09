resource "google_compute_instance" "simulator" {
  count        = var.sim_workers
  name         = "simulator-${count.index}"
  tags         = ["simulator", "singlestore-logistics-sim"]
  machine_type = var.sim_machine_type

  boot_disk {
    initialize_params {
      image = var.machine_image
    }
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
