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

**IMPORTANT**: Since each environment uses a **separate GCP project**, you need **different Service Account keys** for each project.

### Required Secrets

You need to configure **5 separate secrets** in GitHub:

1. **`GCP_SA_KEY_STATE`**: Service Account key for the Terraform state project
2. **`GCP_SA_KEY_DEV`**: Service Account key for the dev environment project
3. **`GCP_SA_KEY_QA`**: Service Account key for the qa environment project
4. **`GCP_SA_KEY_PREPROD`**: Service Account key for the preprod environment project
5. **`GCP_SA_KEY_PROD`**: Service Account key for the prod environment project

### Project Structure

- **State Project**: Dedicated project for Terraform state bucket
- **Dev Project**: Separate project for dev environment resources
- **QA Project**: Separate project for qa environment resources
- **Preprod Project**: Separate project for preprod environment resources
- **Prod Project**: Separate project for prod environment resources

### How to Create Service Accounts

#### 1. State Project Service Account

```bash
# Create Service Account in the state project
gcloud config set project terraform-state-project
gcloud iam service-accounts create terraform-state-sa \
  --display-name="Terraform State Service Account"

# Grant access to state bucket
gcloud projects add-iam-policy-binding terraform-state-project \
  --member="serviceAccount:terraform-state-sa@terraform-state-project.iam.gserviceaccount.com" \
  --role="roles/storage.objectAdmin"

# Create and download key
gcloud iam service-accounts keys create terraform-state-key.json \
  --iam-account=terraform-state-sa@terraform-state-project.iam.gserviceaccount.com
```

#### 2. Environment Project Service Accounts

For each environment project (dev, qa, preprod, prod):

```bash
# Set the environment project
ENV=dev  # or qa, preprod, prod
PROJECT_ID=your-${ENV}-project

gcloud config set project $PROJECT_ID

# Create Service Account
gcloud iam service-accounts create terraform-${ENV}-sa \
  --display-name="Terraform ${ENV} Service Account"

# Grant editor role (or specific roles as needed)
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:terraform-${ENV}-sa@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/editor"

# Grant service account user role for GKE
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:terraform-${ENV}-sa@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/iam.serviceAccountUser"

# Create and download key
gcloud iam service-accounts keys create terraform-${ENV}-key.json \
  --iam-account=terraform-${ENV}-sa@${PROJECT_ID}.iam.gserviceaccount.com
```

**Note**: The state project Service Account also needs read access to the state bucket. The environment Service Accounts only need access to their respective projects.

### Configure Secrets in GitHub

For each Service Account key created:

1. Go to Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Name: `GCP_SA_KEY_STATE`, `GCP_SA_KEY_DEV`, `GCP_SA_KEY_QA`, `GCP_SA_KEY_PREPROD`, or `GCP_SA_KEY_PROD`
4. Value: Complete content of the corresponding JSON key file (copy and paste)

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

- `dev`: Development (1 node) - Deploys to `dev-project`
- `qa`: QA/Testing (2 nodes) - Deploys to `qa-project`
- `preprod`: Pre-production (3 nodes) - Deploys to `preprod-project`
- `prod`: Production (4 nodes) - Deploys to `prod-project`

### tfvars Files

Each workspace has its variables file with its dedicated GCP project:
- `dev.tfvars` - Contains `project_id` for dev-project
- `qa.tfvars` - Contains `project_id` for qa-project
- `preprod.tfvars` - Contains `project_id` for preprod-project
- `prod.tfvars` - Contains `project_id` for prod-project

**Important**: 
- Workspaces must exist beforehand in the Terraform backend. They are not created automatically.
- Each environment uses a **separate GCP project** for its resources
- The Terraform state is stored in a **dedicated state project** (separate from environment projects)

## 🚨 Troubleshooting

### Workflow fails at "Authenticate to Google Cloud"

**Cause**: One or more of the required secrets is not configured or is invalid.

**Solution**:
1. Verify all 5 secrets exist in Settings → Secrets:
   - `GCP_SA_KEY_STATE`
   - `GCP_SA_KEY_DEV`
   - `GCP_SA_KEY_QA`
   - `GCP_SA_KEY_PREPROD`
   - `GCP_SA_KEY_PROD`
2. Verify each JSON is valid
3. Verify each Service Account has the necessary permissions in its respective project
4. Check the workflow logs to see which specific secret/environment failed

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

1. **Credential rotation**: Rotate all Service Account keys periodically (especially prod)
2. **Least privilege principle**: Assign only necessary roles to each Service Account
3. **Separate credentials**: Use different Service Accounts for each environment for better isolation
4. **Plan review**: Always review plans before approving
5. **Protected environments**: Configure more reviewers for prod than for dev
6. **Audit**: Regularly review deployment logs and access patterns
7. **State project security**: Limit access to the state project Service Account

### Minimum Service Account Permissions

Each Service Account needs different permissions:

**State Project Service Account** (`GCP_SA_KEY_STATE`):
- `roles/storage.objectAdmin` - To read/write the state bucket in the state project

**Dev Project Service Account** (`GCP_SA_KEY_DEV`):
- `roles/editor` or specific roles - To create/modify resources in dev-project
- `roles/iam.serviceAccountUser` - To use service accounts in GKE
- `roles/serviceusage.serviceUsageConsumer` - To enable APIs

**QA Project Service Account** (`GCP_SA_KEY_QA`):
- `roles/editor` or specific roles - To create/modify resources in qa-project
- `roles/iam.serviceAccountUser` - To use service accounts in GKE
- `roles/serviceusage.serviceUsageConsumer` - To enable APIs

**Preprod Project Service Account** (`GCP_SA_KEY_PREPROD`):
- `roles/editor` or specific roles - To create/modify resources in preprod-project
- `roles/iam.serviceAccountUser` - To use service accounts in GKE
- `roles/serviceusage.serviceUsageConsumer` - To enable APIs

**Prod Project Service Account** (`GCP_SA_KEY_PROD`):
- `roles/editor` or specific roles - To create/modify resources in prod-project
- `roles/iam.serviceAccountUser` - To use service accounts in GKE
- `roles/serviceusage.serviceUsageConsumer` - To enable APIs

**Note**: Each Service Account only needs permissions in its own project. This provides better security isolation between environments.

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
