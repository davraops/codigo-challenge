# Development Environment Configuration
# IMPORTANT: This project_id is for the DEV environment resources (GKE, VPC, etc.)
# The Terraform state bucket is stored in a SEPARATE project (see backend.tf)
project_id  = "your-project-dev"
region      = "us-central1"
name_prefix = "codigo-dev"
gke_num_nodes = 1

