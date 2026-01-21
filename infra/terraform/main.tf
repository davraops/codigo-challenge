locals {
  cluster_name = "${var.name_prefix}-gke"
}

data "google_client_config" "default" {}
