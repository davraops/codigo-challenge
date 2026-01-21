# Pre-Production Environment Configuration
# IMPORTANT: This project_id is for the PREPROD environment resources (GKE, VPC, etc.)
# The Terraform state bucket is stored in a SEPARATE project (see backend.tf)
project_id  = "your-project-preprod"
region      = "us-central1"
name_prefix = "codigo-preprod"
gke_num_nodes = 3

