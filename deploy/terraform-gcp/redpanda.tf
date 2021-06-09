resource "google_compute_instance" "redpanda" {
  count        = var.rp_nodes
  name         = "rp-node-${count.index}"
  tags         = ["rp-cluster", "singlestore-logistics-sim"]
  machine_type = var.rp_machine_type

  boot_disk {
    initialize_params {
      image = var.machine_image
    }
  }

  dynamic "scratch_disk" {
    for_each = range(var.rp_scratch_disks)
    content {
      interface = "NVME"
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
