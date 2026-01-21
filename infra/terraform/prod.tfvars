# Production Environment Configuration
# IMPORTANT: This project_id is for the PROD environment resources (GKE, VPC, etc.)
# The Terraform state bucket is stored in a SEPARATE project (see backend.tf)
project_id  = "your-project-prod"
region      = "us-central1"
name_prefix = "codigo-prod"
gke_num_nodes = 4

