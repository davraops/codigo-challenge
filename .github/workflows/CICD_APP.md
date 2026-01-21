# Application CI/CD Documentation

This document provides a high-level overview of the CI/CD pipelines for the application components (API and Worker services).

## Overview

The application CI/CD system consists of multiple workflows that handle different stages of the software development lifecycle:

1. **Pull Request Pipelines** - Validate code changes before merging
2. **Push to Main Pipelines** - Build, test, and deploy to development environment
3. **Release Pipelines** - Create versioned releases with security scans
4. **Promotion Pipelines** - Promote releases to higher environments (QA, Preprod, Prod)

## Pipeline Types

### 1. Pull Request Pipelines

**Workflows:** `app-api-pr.yml`, `app-worker-pr.yml`

**Trigger:** Pull requests targeting the `main` branch with changes in `app/api/**` or `app/worker/**`

**Purpose:** Validate code quality and buildability before merging

**Key Activities:**
- PR title validation (Conventional Commits format)
- Code formatting checks
- Linting and static analysis
- Unit tests with coverage reporting
- Type checking (Go build)
- Docker image build verification
- Container health check
- PR comment with results summary

**Runners:** 
- `dedicated-runner` for non-Docker jobs
- `dedicated-runner-dind` for Docker-related jobs

**Output:** PR comments with validation results

---

### 2. Push to Main Pipelines

**Workflows:** `app-api-push.yml`, `app-worker-push.yml`

**Trigger:** Pushes to `main` branch with changes in `app/api/**` or `app/worker/**`

**Purpose:** Build, test, scan, and deploy to the development environment

**Key Activities:**
- Security scans (SAST and SCA) before build
- Docker image build with SHA and `latest` tags
- Container health check
- Integration tests
- Docker image security scan
- Push to GCP Artifact Registry (DEV project)
- Update Kubernetes manifests via GitOps
- Deploy to GKE via ArgoCD
- Deployment verification
- Slack notifications with security and test results

**Environment:** `dev` (with approval protection)

**Output:** 
- Docker images in Artifact Registry
- Deployed application in GKE dev cluster
- Slack notifications

---

### 3. Release Pipelines

**Workflows:** `app-api-release.yml`, `app-worker-release.yml`

**Trigger:** Manual execution (`workflow_dispatch`)

**Inputs:**
- `version`: Release version (e.g., `v1.0.0`)

**Purpose:** Create versioned releases with security validation

**Key Activities:**
- Security scans (SAST and SCA)
- Create Git tag (`{version}-api` or `{version}-worker`)
- Create GitHub Release with security scan results
- Upload scan results to GitHub Security

**Output:**
- Git tags
- GitHub Releases
- Security scan reports in GitHub Security tab

**Note:** No Slack notifications for releases

---

### 4. Promotion Pipelines

**Workflows:** `app-api-promote.yml`, `app-worker-promote.yml`

**Trigger:** Manual execution (`workflow_dispatch`)

**Inputs:**
- `version`: Release version to promote (e.g., `v1.0.0-api` or `v1.0.0-worker`)
- `environment`: Target environment (`qa`, `preprod`, `prod`)

**Purpose:** Promote a released version to higher environments

**Key Activities:**
- **Authorization check** - Verify user is in Release Team
- Checkout code from the specified release tag
- Security scans (SAST and SCA)
- Docker image build with version tag
- Container health check
- Integration tests
- Docker image security scan
- Push to GCP Artifact Registry (target environment project)
- Update Kubernetes manifests via GitOps
- Deploy to GKE via ArgoCD (target environment)
- Deployment verification
- Slack notifications with security and test results

**Environments:** `qa`, `preprod`, `prod` (each with approval protection)

**Authorization:** Only users listed in `RELEASE_TEAM_MEMBERS` variable can execute

**Output:**
- Docker images in Artifact Registry (target environment)
- Deployed application in GKE cluster (target environment)
- Slack notifications

---

## Workflow Architecture

### Pipeline Flow Diagram

```
┌─────────────────┐
│  Developer PR   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  PR Pipeline    │ ◄─── Validation & Testing
│  (Validation)   │
└────────┬────────┘
         │
         ▼ (Merge to main)
┌─────────────────┐
│ Push to Main    │ ◄─── Build, Test, Deploy to DEV
│  Pipeline       │
└────────┬────────┘
         │
         ▼ (Manual Release)
┌─────────────────┐
│ Release Pipeline│ ◄─── Create Versioned Release
│  (Tag & Release)│
└────────┬────────┘
         │
         ▼ (Manual Promotion)
┌─────────────────┐
│ Promotion       │ ◄─── Deploy to QA/PREPROD/PROD
│  Pipeline       │
└─────────────────┘
```

---

## Required Configuration

### GitHub Variables

#### Common Variables
- `GO_VERSION` - Go version (default: `1.22`)
- `POSTGRES_HOST` - PostgreSQL host for tests
- `POSTGRES_PORT` - PostgreSQL port (default: `5432`)
- `POSTGRES_USER` - PostgreSQL user for tests
- `POSTGRES_DB` - PostgreSQL database name for tests
- `NATS_URL` - NATS connection URL
- `GCP_REGION` - GCP region (default: `us-central1`)
- `GCP_ARTIFACT_REGISTRY` - Artifact Registry repository name (default: `codigo-images`)
- `K8S_NAMESPACE` - Kubernetes namespace (default: `codigo`)
- `ARGOCD_NAMESPACE` - ArgoCD namespace (default: `argocd`)
- `ARGOCD_APP_NAME` - ArgoCD application name (default: `codigo-app`)
- `RELEASE_TEAM_MEMBERS` - Comma-separated list of authorized users for promotion (e.g., `user1,user2,user3`)

#### API-Specific Variables
- `API_PORT` - API service port (default: `8080`)
- `API_HEALTH_ENDPOINT` - Health check endpoint (default: `/healthz`)
- `DOCKER_IMAGE_NAME` - Docker image name for API (default: `codigo-api`)
- `K8S_DEPLOYMENT_NAME_API` - Kubernetes deployment name for API (default: `codigo-api`)

#### Worker-Specific Variables
- `DOCKER_IMAGE_NAME_WORKER` - Docker image name for Worker (default: `codigo-worker`)
- `K8S_DEPLOYMENT_NAME_WORKER` - Kubernetes deployment name for Worker (default: `codigo-worker`)
- `WORKER_STARTUP_WAIT_SECONDS` - Worker startup wait time (default: `5`)

#### Environment-Specific Variables
- `GCP_PROJECT_ID_DEV` - GCP project ID for development
- `GCP_PROJECT_ID_QA` - GCP project ID for QA
- `GCP_PROJECT_ID_PREPROD` - GCP project ID for pre-production
- `GCP_PROJECT_ID_PROD` - GCP project ID for production
- `GKE_CLUSTER_NAME` - GKE cluster name (default: `codigo-cluster`)

### GitHub Secrets

#### Common Secrets
- `POSTGRES_PASSWORD` - PostgreSQL password for tests
- `SLACK_WEBHOOK_URL` - Slack webhook URL for notifications
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

#### Environment-Specific Secrets
- `GCP_SA_KEY_DEV` - GCP Service Account key for development
- `GCP_SA_KEY_QA` - GCP Service Account key for QA
- `GCP_SA_KEY_PREPROD` - GCP Service Account key for pre-production
- `GCP_SA_KEY_PROD` - GCP Service Account key for production

---

## Security Features

### Security Scanning

All pipelines include multiple layers of security scanning:

1. **SCA (Software Composition Analysis)** - Trivy scans dependencies in `go.mod` and `go.sum`
   - Detects known vulnerabilities in dependencies
   - Fails build on CRITICAL or HIGH severity issues
   - Results uploaded to GitHub Security

2. **SAST (Static Application Security Testing)** - Gosec scans Go source code
   - Detects security issues in code
   - Results uploaded to GitHub Security

3. **Container Image Scanning** - Trivy scans built Docker images
   - Detects vulnerabilities in container images
   - Results uploaded to GitHub Security

### Authorization

- **Promotion Pipelines**: Only users listed in `RELEASE_TEAM_MEMBERS` can execute
- **Environment Protection**: Each environment (dev, qa, preprod, prod) requires manual approval
- **Branch Protection**: Main branch requires PR approval before merging

---

## Deployment Strategy

### GitOps with ArgoCD

All deployments use GitOps principles:

1. **Manifest Updates**: CI/CD pipelines update `k8s/apps/codigo/values.yaml` with new image tags
2. **Git Commit**: Changes are committed and pushed to the `main` branch
3. **ArgoCD Sync**: ArgoCD automatically detects changes and syncs to Kubernetes
4. **Verification**: Pipelines verify deployment status in GKE

### Image Tagging Strategy

- **Push to Main**: Images tagged with `{SHA}` and `latest`
- **Releases**: Images tagged with `{version}-api` or `{version}-worker`
- **Promotions**: Images tagged with `{version}` and `latest` in target environment registry

---

## Notification System

### Slack Notifications

Slack notifications are sent for:

- **Push to Main**: Deployment started, success, or failure
- **Promotion**: Deployment started, success, or failure

Each notification includes:
- Service name and environment
- Version/commit information
- Security scan results (SCA, SAST)
- Integration test results
- Link to workflow run

**Note:** Release pipelines do not send Slack notifications.

---

## Best Practices

### Code Quality

- All code must pass PR validation before merging
- Format checks, linting, and tests are mandatory
- Coverage reports are generated and uploaded

### Security

- Security scans run before and after Docker build
- Critical/high vulnerabilities block deployments
- All scan results are tracked in GitHub Security

### Deployment

- Development deployments are automatic after merge
- Higher environment deployments require manual promotion
- All deployments are verified before completion
- GitOps ensures infrastructure state matches Git

### Release Management

- Releases are created manually with version tags
- Releases include security validation
- Promotions require Release Team authorization
- Each environment has separate GCP projects and credentials

---

## Troubleshooting

### Common Issues

**Pipeline fails at authorization check:**
- Verify user is listed in `RELEASE_TEAM_MEMBERS` variable
- Check variable is set correctly (comma-separated, no spaces)

**Docker build fails:**
- Check Dockerfile syntax
- Verify all dependencies are available
- Check Docker daemon is running (for local testing)

**Deployment fails:**
- Verify GCP credentials are correct for target environment
- Check GKE cluster is accessible
- Verify ArgoCD is properly configured
- Check Kubernetes manifests are valid

**Security scans fail:**
- Review vulnerabilities in GitHub Security tab
- Update dependencies if needed
- Address code security issues reported by SAST

**Slack notifications not working:**
- Verify `SLACK_WEBHOOK_URL` secret is set
- Check webhook URL is valid and active
- Review Slack app permissions

---

## Workflow Execution Summary

| Pipeline Type | Trigger | Environment | Authorization | Slack Notifications |
|--------------|---------|-------------|---------------|---------------------|
| PR | Pull Request | N/A | All users | No |
| Push to Main | Push to main | dev | All users | Yes |
| Release | Manual | N/A | All users | No |
| Promotion | Manual | qa/preprod/prod | Release Team only | Yes |

---

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [Trivy Documentation](https://aquasecurity.github.io/trivy/)
- [Gosec Documentation](https://github.com/securego/gosec)

---

**Last Updated:** 2024

