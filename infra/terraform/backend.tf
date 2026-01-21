# Terraform Backend Configuration
# This file defines a GCS backend for storing Terraform state remotely
# 
# IMPORTANT: The GCS bucket must be created MANUALLY before using this backend.
# 
# To create the bucket manually:
# gcloud storage buckets create gs://YOUR-PROJECT-ID-terraform-state \
#   --location=us-central1 \
#   --uniform-bucket-level-access
# gcloud storage buckets update gs://YOUR-PROJECT-ID-terraform-state --versioning
#
# Then update the bucket name below and run: terraform init

terraform {
  backend "gcs" {
    bucket = "REPLACE_WITH_BUCKET_NAME"  # e.g., "your-project-id-terraform-state"
    prefix = "terraform/state"  # Workspace name will be appended automatically
  }
}

