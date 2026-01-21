# Terraform - GCP + GKE

This Terraform configuration provisions all required GCP resources for the Codigo SRE challenge, including GKE cluster, VPC networking, Artifact Registry, and IAM service accounts.

## Prerequisites

- `gcloud` CLI installed and authenticated
- Terraform >= 1.5.0
- A GCP project with billing enabled
- Your GCP account has `roles/owner` or `roles/editor` on the project

**Note**: All required GCP APIs are enabled automatically by this Terraform configuration, so no manual API enablement is needed.

## Multi-Environment Strategy

This configuration uses **Terraform Workspaces** combined with environment-specific `tfvars` files to manage multiple environments (dev, qa, preprod, prod).

### Workspace Benefits

- **Isolated State**: Each environment has its own state file in the backend
- **Environment Separation**: Prevents accidental changes across environments
- **Easy Switching**: Switch between environments with a single command
- **Shared Configuration**: Same Terraform code, different variable values

### Available Environments

- `dev`: Development environment (1 node, lower resources)
- `qa`: QA/Testing environment (2 nodes)
- `preprod`: Pre-production environment (3 nodes)
- `prod`: Production environment (4 nodes)

## Quick Start

### 1. Configure Backend

This configuration uses a GCS backend for remote state storage.

1. Create the GCS bucket manually (one-time setup):
```bash
# Replace YOUR-PROJECT-ID with your actual GCP project ID
gcloud storage buckets create gs://YOUR-PROJECT-ID-terraform-state \
  --location=us-central1 \
  --uniform-bucket-level-access

# Enable versioning (recommended for state file history)
gcloud storage buckets update gs://YOUR-PROJECT-ID-terraform-state \
  --versioning
```

2. Update `backend.tf` with your bucket name:
```hcl
terraform {
  backend "gcs" {
    bucket = "YOUR-PROJECT-ID-terraform-state"  # Use your actual bucket name
    prefix = "terraform/state"
  }
}
```

3. Initialize Terraform with the backend:
```bash
terraform init
```

### 2. Initialize Workspaces

Create and select a workspace for your environment:

```bash
cd infra/terraform

# For Development
terraform workspace new dev
terraform workspace select dev

# For QA
terraform workspace new qa
terraform workspace select qa

# For Pre-Production
terraform workspace new preprod
terraform workspace select preprod

# For Production
terraform workspace new prod
terraform workspace select prod
```

**Note**: Workspaces are created once and persist in the backend. Use `terraform workspace select <name>` to switch between them.

### 3. Configure Environment Variables

Each environment has its own `tfvars` file. Update the appropriate file with your project details:

- `dev.tfvars` - Development environment
- `qa.tfvars` - QA environment
- `preprod.tfvars` - Pre-production environment
- `prod.tfvars` - Production environment

Example for dev:
```hcl
project_id  = "your-project-dev"
region      = "us-central1"
name_prefix = "codigo-dev"
gke_num_nodes = 1
```

### 4. Apply Infrastructure

```bash
# Ensure you're in the correct workspace
terraform workspace show  # Verify current workspace

# Plan with environment-specific tfvars
terraform plan -var-file=dev.tfvars      # For dev
terraform plan -var-file=qa.tfvars      # For qa
terraform plan -var-file=preprod.tfvars  # For preprod
terraform plan -var-file=prod.tfvars    # For prod

# Apply
terraform apply -var-file=dev.tfvars      # For dev
terraform apply -var-file=qa.tfvars       # For qa
terraform apply -var-file=preprod.tfvars  # For preprod
terraform apply -var-file=prod.tfvars     # For prod
```

### 5. Switch Between Environments

```bash
# List all workspaces
terraform workspace list

# Switch to a different environment
terraform workspace select dev
terraform workspace select qa
terraform workspace select preprod
terraform workspace select prod

# Show current workspace
terraform workspace show
```

### 6. Connect to Cluster

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

**Note**: The Terraform state bucket (if using remote state) is not managed by Terraform and must be deleted manually if needed.

**Note**: GCP APIs will remain enabled (they don't incur costs when not in use). To disable them manually:
```bash
gcloud services disable container.googleapis.com artifactregistry.googleapis.com compute.googleapis.com
```

## Workspace Management

### Common Workspace Commands

```bash
# List all workspaces
terraform workspace list

# Show current workspace
terraform workspace show

# Create a new workspace
terraform workspace new <name>

# Select a workspace
terraform workspace select <name>

# Delete a workspace (be careful!)
terraform workspace delete <name>
```

### State File Organization

With workspaces, state files are stored separately in the backend:
- `terraform/state/dev/terraform.tfstate`
- `terraform/state/qa/terraform.tfstate`
- `terraform/state/preprod/terraform.tfstate`
- `terraform/state/prod/terraform.tfstate`

This ensures complete isolation between environments.

### Best Practices

1. **Always verify workspace**: Run `terraform workspace show` before applying
2. **Use correct tfvars**: Match the tfvars file to your current workspace
3. **Separate projects**: Use different GCP projects for each environment when possible
4. **Lock state**: The GCS backend provides state locking automatically
5. **Review before apply**: Always run `terraform plan` before `terraform apply`

## Troubleshooting

- **API errors**: Wait a few minutes after first apply for APIs to fully enable
- **Permission errors**: Ensure your account has `roles/owner` or `roles/editor`
- **Billing errors**: Verify billing is enabled on your GCP project
- **Wrong workspace**: Always verify with `terraform workspace show` before applying
- **State conflicts**: Ensure you're using the correct workspace and tfvars file
