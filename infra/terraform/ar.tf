resource "google_artifact_registry_repository" "repo" {
  location      = var.region
  repository_id = "${var.name_prefix}-repo"
  format        = "DOCKER"
}
