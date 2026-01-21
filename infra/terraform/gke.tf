resource "google_container_cluster" "gke" {
  name     = local.cluster_name
  location = var.region

  network    = google_compute_network.vpc.id
  subnetwork = google_compute_subnetwork.subnet.id

  remove_default_node_pool = true
  initial_node_count       = 1

  ip_allocation_policy {
    cluster_secondary_range_name  = "pods"
    services_secondary_range_name = "services"
  }

  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  depends_on = [
    google_project_service.required_apis
  ]
}

resource "google_container_node_pool" "primary_nodes" {
  name       = "${var.name_prefix}-np"
  location   = var.region
  cluster    = google_container_cluster.gke.name
  node_count = var.gke_num_nodes

  node_config {
    machine_type    = "e2-standard-2"
    service_account = google_service_account.gke_nodes.email
    oauth_scopes    = ["https://www.googleapis.com/auth/cloud-platform"]

    labels = {
      env = "takehome"
    }
  }
}
