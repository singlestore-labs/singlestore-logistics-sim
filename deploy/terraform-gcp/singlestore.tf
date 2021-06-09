resource "google_compute_instance" "singlestore_agg" {
  count        = var.s2_aggs
  name         = "s2-agg-${count.index}"
  tags         = ["s2-cluster", "s2-agg", "singlestore-logistics-sim"]
  machine_type = var.s2_machine_type

  boot_disk {
    initialize_params {
      image = var.machine_image
      size  = 64
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

resource "google_compute_instance" "singlestore_leaf" {
  count        = var.s2_leaves
  name         = "s2-leaf-${count.index}"
  tags         = ["s2-cluster", "s2-leaf", "singlestore-logistics-sim"]
  machine_type = var.s2_machine_type

  boot_disk {
    initialize_params {
      image = var.machine_image
      size  = 64
    }
  }

  dynamic "scratch_disk" {
    for_each = range(var.s2_scratch_disks)
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
