# Enable required GCP APIs
# This ensures the infrastructure works from a fresh GCP project

resource "google_project_service" "required_apis" {
  for_each = toset([
    "container.googleapis.com",      # GKE
    "artifactregistry.googleapis.com", # Artifact Registry
    "compute.googleapis.com",         # VPC, networking
    "iam.googleapis.com",             # Service accounts
    "logging.googleapis.com",         # Cloud Logging
    "monitoring.googleapis.com",      # Cloud Monitoring
  ])

  project = var.project_id
  service = each.value

  disable_on_destroy = false
}

