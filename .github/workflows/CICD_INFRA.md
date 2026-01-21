# Terraform CI/CD Pipeline

This directory contains GitHub Actions workflows for automated validation and deployment of infrastructure using Terraform.

## 📋 Workflows

### 1. Terraform PR Validation (`terraform-pr.yml`)

**Trigger**: Automatically runs when a Pull Request is opened against `main` with changes in `infra/terraform/**`

**Purpose**: Validate Terraform changes before merging

**Flow**:
1. **Single Validation**: Runs `terraform fmt` and `terraform validate` once
2. **Parallel Plans**: Executes `terraform plan` for each workspace (dev, qa, preprod, prod) in parallel
3. **PR Comment**: Posts a comment with results for each workspace

**Features**:
- ✅ Format and syntax validation
- ✅ Plans executed in parallel for all workspaces
- ✅ Automatic comments on PR with results
- ✅ Fails if any plan fails

### 2. Terraform Apply (`terraform-apply.yml`)

**Trigger**: Automatically runs when pushing to `main` with changes in `infra/terraform/**`

**Purpose**: Apply infrastructure changes after merge

**Flow**:
1. **Validation**: Runs `terraform fmt` and `terraform validate`
2. **Parallel Plans**: Executes `terraform plan` for each workspace in parallel
3. **Approval Required**: Each workspace requires manual approval before applying
4. **Apply**: Executes `terraform apply` only if the corresponding plan was successful

**Features**:
- ✅ Validation before applying
- ✅ Parallel plans
- ✅ **Manual approval required** for each environment (dev, qa, preprod, prod)
- ✅ Apply only executes if plan was successful
- ✅ Uses artifacts to pass plan file to apply

## 🔐 Required Secrets

### GCP_SA_KEY (Required)

**Description**: GCP Service Account JSON key with permissions for:
- Read/write to Terraform state bucket (GCS)
- Create/modify GCP resources (GKE, VPC, IAM, etc.)

**How to obtain**:
```bash
# Create Service Account
gcloud iam service-accounts create terraform-ci \
  --display-name="Terraform CI/CD"

# Assign required roles
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:terraform-ci@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/editor"

# Create and download key
gcloud iam service-accounts keys create terraform-ci-key.json \
  --iam-account=terraform-ci@PROJECT_ID.iam.gserviceaccount.com
```

**Configure in GitHub**:
1. Go to Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Name: `GCP_SA_KEY`
4. Value: Complete content of the JSON file (copy and paste)

## 🌍 Environments

Environments in GitHub Actions allow requiring manual approval before applying changes.

### Environment Setup

1. Go to **Settings → Environments** in your repository
2. Create the following environments:
   - `dev`
   - `qa`
   - `preprod`
   - `prod`

### Configuration per Environment

For each environment, configure:

#### Required Reviewers
- **dev**: Optional (can auto-approve or require 1 reviewer)
- **qa**: Requires 1 reviewer
- **preprod**: Requires 1-2 reviewers
- **prod**: Requires 2+ reviewers (maximum security)

#### Deployment Branches
- Configure to allow only `main` branch

#### Protection Rules (Optional)
- Timeout: Define maximum wait time for approval
- Wait timer: Wait time before allowing approval (useful for prod)

### Configuration Example

```
Environment: prod
├── Required reviewers: 2
├── Deployment branches: main only
└── Wait timer: 5 minutes
```

## 🔄 Complete Flow

### Pull Request Flow

```
Developer creates PR with changes in infra/terraform/
         ↓
terraform-pr.yml triggers
         ↓
┌─────────────────────────────────┐
│ 1. terraform-validate            │
│    - Format check                │
│    - Validate                    │
└─────────────────────────────────┘
         ↓
┌─────────────────────────────────┐
│ 2. terraform-plan (parallel)    │
│    - Plan dev                   │
│    - Plan qa                    │
│    - Plan preprod               │
│    - Plan prod                  │
└─────────────────────────────────┘
         ↓
┌─────────────────────────────────┐
│ 3. Comment on PR                │
│    - Results per workspace      │
└─────────────────────────────────┘
```

### Merge to Main Flow

```
PR is approved and merged to main
         ↓
terraform-apply.yml triggers
         ↓
┌─────────────────────────────────┐
│ 1. terraform-validate            │
│    - Format check                │
│    - Validate                    │
└─────────────────────────────────┘
         ↓
┌─────────────────────────────────┐
│ 2. terraform-plan (parallel)    │
│    - Plan dev → artifact        │
│    - Plan qa → artifact         │
│    - Plan preprod → artifact    │
│    - Plan prod → artifact       │
└─────────────────────────────────┘
         ↓
┌─────────────────────────────────┐
│ 3. Wait for manual approval     │
│    ⏸️  Paused for approval      │
│    - dev (optional)             │
│    - qa (requires approval)     │
│    - preprod (requires approv.) │
│    - prod (requires approv.)    │
└─────────────────────────────────┘
         ↓
┌─────────────────────────────────┐
│ 4. terraform-apply (parallel)   │
│    - Apply dev (if approved)    │
│    - Apply qa (if approved)     │
│    - Apply preprod (if approved)│
│    - Apply prod (if approved)   │
└─────────────────────────────────┘
```

## 📝 Workspaces and Files

### Configured Workspaces

- `dev`: Development (1 node)
- `qa`: QA/Testing (2 nodes)
- `preprod`: Pre-production (3 nodes)
- `prod`: Production (4 nodes)

### tfvars Files

Each workspace has its variables file:
- `dev.tfvars`
- `qa.tfvars`
- `preprod.tfvars`
- `prod.tfvars`

**Important**: Workspaces must exist beforehand in the Terraform backend. They are not created automatically.

## 🚨 Troubleshooting

### Workflow fails at "Authenticate to Google Cloud"

**Cause**: The `GCP_SA_KEY` secret is not configured or is invalid.

**Solution**:
1. Verify the secret exists in Settings → Secrets
2. Verify the JSON is valid
3. Verify the Service Account has the necessary permissions

### Plan fails with "Workspace does not exist"

**Cause**: The workspace does not exist in the Terraform backend.

**Solution**:
```bash
cd infra/terraform
terraform init
terraform workspace new dev  # or qa, preprod, prod
```

### Apply does not run even though plan was successful

**Cause**: The plan artifact was not downloaded correctly or the plan failed.

**Solution**:
1. Check the logs of the `terraform-plan` job for the specific workspace
2. Verify the artifact was uploaded correctly
3. Verify the apply job is waiting for environment approval

### Cannot approve deployment

**Cause**: You don't have permissions or are not configured as a reviewer.

**Solution**:
1. Verify you are in the required reviewers list for the environment
2. Verify you have write permissions in the repository
3. Contact the repository administrator

## 🔒 Security

### Best Practices

1. **Credential rotation**: Rotate `GCP_SA_KEY` periodically
2. **Least privilege principle**: Assign only necessary roles to the Service Account
3. **Plan review**: Always review plans before approving
4. **Protected environments**: Configure more reviewers for prod than for dev
5. **Audit**: Regularly review deployment logs

### Minimum Service Account Permissions

The Service Account needs these roles:
- `roles/storage.objectAdmin` - To access the state bucket
- `roles/editor` or specific roles - To create/modify resources
- `roles/iam.serviceAccountUser` - To use service accounts in GKE

## 📊 Monitoring

### View Workflow Status

1. Go to the **Actions** tab in GitHub
2. Select the workflow you want to view
3. Review the logs of each job

### Notifications

Configure notifications in GitHub for:
- Workflow failures
- Deployment approval requests
- Deployment completions

## 🔧 Customization

### Add New Workspaces

1. Create the corresponding `tfvars` file (e.g., `staging.tfvars`)
2. Add the workspace to the matrix in both workflows:
   ```yaml
   strategy:
     matrix:
       workspace: [dev, qa, preprod, prod, staging]
   ```
3. Create the workspace in Terraform:
   ```bash
   terraform workspace new staging
   ```
4. Create the environment in GitHub Settings

### Change Terraform Version

Edit the `TF_VERSION` variable in both workflows:
```yaml
env:
  TF_VERSION: '1.5.0'  # Change here
```

## 📚 References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Terraform Documentation](https://www.terraform.io/docs)
- [GitHub Environments](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment)
