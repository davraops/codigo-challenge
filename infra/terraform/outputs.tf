output "cluster_name" {
  value = google_container_cluster.gke.name
}

output "cluster_region" {
  value = var.region
}

output "artifact_registry_repo" {
  value = google_artifact_registry_repository.repo.repository_id
}

output "connect_command" {
  value = "gcloud container clusters get-credentials ${google_container_cluster.gke.name} --region ${var.region} --project ${var.project_id}"
}
