# Terraform - GCP + GKE

This Terraform configuration provisions all required GCP resources for the Codigo SRE challenge, including GKE cluster, VPC networking, Artifact Registry, and IAM service accounts.

## Prerequisites

- `gcloud` CLI installed and authenticated
- Terraform >= 1.5.0
- A GCP project with billing enabled
- Your GCP account has `roles/owner` or `roles/editor` on the project

**Note**: All required GCP APIs are enabled automatically by this Terraform configuration, so no manual API enablement is needed.

## Quick Start

### 1. Configure Variables

Create a `terraform.tfvars` file:

```hcl
project_id  = "your-gcp-project-id"
region      = "us-central1"  # optional, defaults to us-central1
name_prefix = "codigo-sre"   # optional, defaults to codigo-sre
gke_num_nodes = 2            # optional, defaults to 2
```

### 2. Initialize and Apply

```bash
cd infra/terraform
terraform init
terraform plan   # Review what will be created
terraform apply  # Creates all resources
```

### 3. Connect to Cluster

After apply completes, connect to your GKE cluster:

```bash
$(terraform output -raw connect_command)
```

Or manually:
```bash
gcloud container clusters get-credentials $(terraform output -raw cluster_name) \
  --region $(terraform output -raw cluster_region) \
  --project $(terraform output -raw project_id)
```

Verify cluster access:
```bash
kubectl get nodes
```

## Resources Created

- **GKE Cluster**: Kubernetes cluster with Workload Identity enabled
- **VPC Network**: Custom VPC with subnet and secondary IP ranges for pods/services
- **Node Pool**: GKE node pool with e2-standard-2 instances
- **Artifact Registry**: Docker repository for container images
- **Service Account**: GKE nodes service account with logging/monitoring permissions
- **APIs**: All required GCP APIs are automatically enabled

## Outputs

- `cluster_name`: GKE cluster name
- `cluster_region`: GCP region where cluster is located
- `artifact_registry_repo`: Artifact Registry repository ID
- `connect_command`: Command to get kubeconfig and connect to cluster

## Cleanup

To destroy all resources and avoid ongoing charges:

```bash
terraform destroy
```

This will remove:
- GKE cluster and node pool
- VPC network and subnet
- Artifact Registry repository
- Service account and IAM bindings

**Note**: GCP APIs will remain enabled (they don't incur costs when not in use). To disable them manually:
```bash
gcloud services disable container.googleapis.com artifactregistry.googleapis.com compute.googleapis.com
```

## Troubleshooting

- **API errors**: Wait a few minutes after first apply for APIs to fully enable
- **Permission errors**: Ensure your account has `roles/owner` or `roles/editor`
- **Billing errors**: Verify billing is enabled on your GCP project
