# Terraform Backend Configuration
# This file defines a GCS backend for storing Terraform state remotely
# 
# IMPORTANT: 
# - The GCS bucket must be created MANUALLY in a SEPARATE GCP project dedicated to state storage
# - This bucket stores state for ALL environments (dev, qa, preprod, prod)
# - Each environment's resources are deployed in their own GCP project (see *.tfvars files)
# 
# Project Structure:
# - terraform-state-project: Contains the GCS bucket for Terraform state
# - dev-project: Contains dev environment resources (GKE, VPC, etc.)
# - qa-project: Contains qa environment resources
# - preprod-project: Contains preprod environment resources
# - prod-project: Contains prod environment resources
#
# To create the bucket manually in the state project:
# gcloud config set project TERRAFORM-STATE-PROJECT-ID
# gcloud storage buckets create gs://TERRAFORM-STATE-PROJECT-ID-terraform-state \
#   --location=us-central1 \
#   --uniform-bucket-level-access
# gcloud storage buckets update gs://TERRAFORM-STATE-PROJECT-ID-terraform-state --versioning
#
# Then update the bucket name below and run: terraform init

terraform {
  backend "gcs" {
    bucket = "REPLACE_WITH_BUCKET_NAME"  # e.g., "terraform-state-project-terraform-state"
    prefix = "terraform/state"  # Workspace name will be appended automatically
  }
}

