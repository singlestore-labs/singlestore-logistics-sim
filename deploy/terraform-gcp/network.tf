resource "google_compute_network" "vpc_network" {
  name                    = "singlestore-logistics-network"
  auto_create_subnetworks = true
}

resource "google_compute_firewall" "external" {
  name    = "external-access"
  network = google_compute_network.vpc_network.self_link

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
}

resource "google_compute_firewall" "internal" {
  name    = "internal-access"
  network = google_compute_network.vpc_network.self_link

  source_tags = ["singlestore-logistics-sim"]

  allow {
    protocol = "all"
  }
}
