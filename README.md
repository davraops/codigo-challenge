# Codigo SRE Take-Home Project

A comprehensive SRE implementation featuring Infrastructure as Code, GitOps deployment, observability stack, CI/CD pipelines, security hardening, and operational tooling.

## 🏗️ Architecture Overview

This project implements a production-ready SRE platform with:

- **Infrastructure**: GKE cluster with multi-environment support (dev, qa, preprod, prod)
- **GitOps**: ArgoCD with App-of-Apps pattern for automated deployments
- **Observability**: Prometheus, Grafana, Loki, Tempo, and OpenTelemetry
- **CI/CD**: GitHub Actions pipelines for infrastructure and application
- **Security**: RBAC, pod security hardening, secrets management
- **Monitoring**: SLIs, SLOs, alerting rules, and runbooks
- **Cost Awareness**: Cost optimization and monitoring strategies
- **Automation**: SLO reporter tool for error budget tracking

## 📁 Project Structure

```
codigo-challenge/
├── app/                          # Application code
│   ├── api/                      # Go API service
│   └── worker/                   # Go Worker service
├── infra/                        # Infrastructure as Code
│   └── terraform/               # Terraform configurations
│       ├── README.md            # Terraform setup guide
│       ├── *.tf                # Terraform resources
│       └── *.tfvars             # Environment-specific variables
├── k8s/                          # Kubernetes manifests
│   ├── apps/                    # Application Helm charts
│   │   └── codigo/              # Main application chart
│   └── observability/           # Observability stack Helm chart
├── argocd/                       # ArgoCD GitOps configuration
│   ├── README.md                # ArgoCD setup guide
│   ├── bootstrap/               # Root application
│   └── apps/                    # App-of-Apps pattern
├── .github/workflows/            # CI/CD pipelines
│   ├── CICD_INFRA.md            # Infrastructure CI/CD docs
│   ├── CICD_APP.md              # Application CI/CD docs
│   ├── terraform-pr.yml         # Terraform PR validation
│   ├── terraform-apply.yml        # Terraform deployment
│   ├── app-api-pr.yml          # API PR pipeline
│   ├── app-worker-pr.yml       # Worker PR pipeline
│   ├── app-api-push.yml        # API deployment pipeline
│   ├── app-worker-push.yml     # Worker deployment pipeline
│   ├── app-api-release.yml     # API release pipeline
│   ├── app-worker-release.yml  # Worker release pipeline
│   ├── app-api-promote.yml     # API promotion pipeline
│   └── app-worker-promote.yml  # Worker promotion pipeline
├── tools/                        # Operational tools
│   └── slo-reporter/            # SLO tracking tool
│       ├── README.md           # Tool documentation
│       └── main.go             # Go implementation
└── Documentation
    ├── OBSERVABILITY.md         # Observability stack guide
    ├── SECURITY.md              # Security implementation
    ├── SEC_BASELINE.md          # Security baseline summary
    ├── SLO_SLI_ALERTS.md        # SLIs, SLOs, and alerting
    ├── COST.md                  # Cost analysis and optimization
    └── AI_NOTES.md              # AI usage documentation
```

## 🚀 Quick Start

### Prerequisites

- GCP account with billing enabled
- `gcloud` CLI installed and configured
- `terraform` >= 1.5.0
- `kubectl` installed
- `helm` >= 3.0 (optional, for local testing)
- Go 1.22+ (for building tools)

### 1. Infrastructure Setup

**Create GCP Projects:**
- 1 project for Terraform state (GCS bucket)
- 4 projects for environments (dev, qa, preprod, prod)

**Provision Infrastructure:**

```bash
cd infra/terraform

# Create GCS bucket for Terraform state (manual step)
# See: infra/terraform/README.md

# Initialize Terraform
terraform init

# Create workspace and apply for dev environment
terraform workspace new dev
terraform workspace select dev
terraform apply -var-file=dev.tfvars

# Repeat for qa, preprod, prod
```

**Detailed Instructions:** See [Infrastructure README](infra/terraform/README.md)

### 2. ArgoCD Setup

```bash
# Install ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server -n argocd --timeout=300s

# Bootstrap GitOps
# Update REPO_URL in argocd/bootstrap/root-app.yaml
kubectl apply -n argocd -f argocd/bootstrap/root-app.yaml
```

**Detailed Instructions:** See [ArgoCD README](argocd/README.md)

### 3. Verify Deployment

```bash
# Check ArgoCD applications
kubectl get applications -n argocd

# Access Grafana
kubectl port-forward svc/kube-prometheus-stack-grafana -n observability 3000:80
# Open http://localhost:3000 (admin/admin)

# Access Prometheus
kubectl port-forward svc/kube-prometheus-stack-prometheus -n observability 9090:9090
# Open http://localhost:9090
```

## 📚 Documentation

### Core Documentation

- **[OBSERVABILITY.md](OBSERVABILITY.md)** - Complete observability stack documentation
  - Metrics, logs, and traces setup
  - Grafana dashboards
  - Prometheus, Loki, Tempo configuration
  - Troubleshooting guide

- **[SECURITY.md](SECURITY.md)** - Security implementation guide
  - Secrets management
  - RBAC configuration
  - Pod security hardening
  - Best practices

- **[SEC_BASELINE.md](SEC_BASELINE.md)** - Security baseline summary
  - Quick reference for security requirements
  - Compliance checklist
  - Verification commands

- **[SLO_SLI_ALERTS.md](SLO_SLI_ALERTS.md)** - SLIs, SLOs, and alerting
  - Service Level Indicators
  - Service Level Objectives
  - Alerting rules
  - Runbooks

- **[COST.md](COST.md)** - Cost awareness and optimization
  - Cost drivers analysis
  - Optimization strategies
  - Cost monitoring recommendations
  - Estimated savings potential

- **[AI_NOTES.md](AI_NOTES.md)** - AI tool usage documentation
  - Tools used
  - Tasks assisted
  - Manual verification process

### Infrastructure Documentation

- **[Infrastructure README](infra/terraform/README.md)** - Terraform setup and usage
  - Multi-environment strategy
  - Backend configuration
  - Resource provisioning
  - Cleanup instructions

### CI/CD Documentation

- **[Infrastructure CI/CD](.github/workflows/CICD_INFRA.md)** - Terraform CI/CD pipelines
  - PR validation workflow
  - Apply workflow
  - Required secrets and variables
  - Multi-project authentication

- **[Application CI/CD](.github/workflows/CICD_APP.md)** - Application CI/CD pipelines
  - PR pipelines
  - Push to main pipelines
  - Release pipelines
  - Promotion pipelines

### GitOps Documentation

- **[ArgoCD README](argocd/README.md)** - GitOps setup and configuration
  - App-of-Apps pattern
  - Application definitions
  - Sync policies
  - Troubleshooting

### Tools Documentation

- **[SLO Reporter README](tools/slo-reporter/README.md)** - SLO tracking tool
  - Installation and usage
  - Integration examples
  - Prometheus queries

## 🔧 Key Features

### Infrastructure

- ✅ **Multi-Environment Support**: Separate GCP projects for dev, qa, preprod, prod
- ✅ **Terraform Workspaces**: Environment-specific state management
- ✅ **GCS Backend**: Remote state storage
- ✅ **GKE Cluster**: Managed Kubernetes with Workload Identity
- ✅ **Artifact Registry**: Container image storage

### CI/CD

- ✅ **Terraform Pipelines**: Automated validation and deployment
- ✅ **Application Pipelines**: PR validation, deployment, release, promotion
- ✅ **Security Scans**: SAST (Gosec) and SCA (Trivy) integration
- ✅ **GitOps Integration**: Automated ArgoCD deployments
- ✅ **Slack Notifications**: Deployment status updates

### Observability

- ✅ **Metrics**: Prometheus with ServiceMonitors
- ✅ **Logs**: Loki with Promtail
- ✅ **Traces**: Tempo with OpenTelemetry
- ✅ **Dashboards**: Grafana with Golden Signals and Dependencies
- ✅ **Instrumentation**: Structured logs, metrics, and traces in code

### Security

- ✅ **No Plaintext Secrets**: All secrets managed via Kubernetes Secrets
- ✅ **RBAC**: ServiceAccounts with namespace isolation
- ✅ **Pod Hardening**: Non-root, read-only filesystem, minimal capabilities
- ✅ **Security Standards**: Compliant with Restricted Pod Security Standard

### Monitoring

- ✅ **SLIs**: Availability, Latency p95, Job Success Rate
- ✅ **SLOs**: 99.9% availability, p95 ≤ 500ms
- ✅ **Alerting**: PrometheusRule with clear thresholds
- ✅ **Runbooks**: Detailed incident response procedures

### Cost Optimization

- ✅ **Resource Right-Sizing**: Optimized requests and limits
- ✅ **Storage Retention**: Configurable retention periods
- ✅ **Multi-Environment Strategy**: Cost isolation per environment
- ✅ **Cost Monitoring**: Recommendations for production

### Automation

- ✅ **SLO Reporter**: Command-line tool for error budget tracking
- ✅ **Prometheus Integration**: Direct queries for SLO calculation
- ✅ **Multiple Output Formats**: Text and JSON

## 🛠️ Tools

### SLO Reporter

A Go-based tool for tracking Service Level Objectives:

```bash
cd tools/slo-reporter
make build
./slo-reporter -prometheus-url http://localhost:9090
```

See [SLO Reporter README](tools/slo-reporter/README.md) for details.

## 🔐 Security

### Secrets Management

**⚠️ IMPORTANT:** Before deployment, create all required secrets:

```bash
# PostgreSQL password
kubectl create secret generic postgres-secret \
  --from-literal=POSTGRES_PASSWORD=<secure-password> \
  -n codigo

# GCP Service Account keys (for CI/CD)
# See: .github/workflows/CICD_INFRA.md
```

See [SECURITY.md](SECURITY.md) for complete security guide.

## 📊 Monitoring

### Dashboards

- **Golden Signals**: Latency, Traffic, Errors, Saturation
- **Dependencies**: PostgreSQL and NATS metrics

Access Grafana: `kubectl port-forward svc/kube-prometheus-stack-grafana -n observability 3000:80`

### SLOs

- **Availability**: 99.9% target
- **Latency**: p95 ≤ 500ms target

Track SLOs: Use the [SLO Reporter](tools/slo-reporter/) tool

See [SLO_SLI_ALERTS.md](SLO_SLI_ALERTS.md) for complete SLO definitions.

## 💰 Cost Management

**Estimated Monthly Cost:** ~$646/month (all environments)

**Optimization Potential:** 40-60% reduction with autoscaling and right-sizing

See [COST.md](COST.md) for detailed cost analysis and optimization strategies.

## 🚨 Alerts

Alerting rules are defined in `k8s/apps/codigo/templates/prometheusrule-slo.yaml`:

- High Error Rate (Warning)
- High Latency (Warning)
- Critical Error Rate (Critical)
- Job Processing Failure (Warning)

Runbooks available in [SLO_SLI_ALERTS.md](SLO_SLI_ALERTS.md).

## 🔄 CI/CD Workflows

### Infrastructure

- **PR Pipeline**: Validates and plans for all environments
- **Apply Pipeline**: Deploys with manual approval per environment

See [Infrastructure CI/CD](.github/workflows/CICD_INFRA.md).

### Application

- **PR Pipelines**: Validation, testing, Docker build, health check
- **Push Pipeline**: Build, test, scan, deploy to dev
- **Release Pipeline**: Create versioned releases
- **Promotion Pipeline**: Promote releases to qa/preprod/prod

See [Application CI/CD](.github/workflows/CICD_APP.md).

## 🧪 Testing

### Run SLO Reporter

```bash
# Port-forward Prometheus
kubectl port-forward svc/kube-prometheus-stack-prometheus -n observability 9090:9090

# Run SLO reporter
cd tools/slo-reporter
./slo-reporter -prometheus-url http://localhost:9090
```

### Verify Security

```bash
# Check non-root users
kubectl get pods -n codigo -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.securityContext.runAsUser}{"\n"}{end}'

# Verify read-only filesystem
kubectl exec -it deployment/codigo-api -n codigo -- touch /test
# Should fail with "read-only file system"
```

## 📝 Requirements Checklist

- ✅ Infrastructure as Code (Terraform)
- ✅ GitOps deployment (ArgoCD)
- ✅ Observability (Metrics, Logs, Traces)
- ✅ CI/CD pipelines
- ✅ Security baseline
- ✅ SLIs, SLOs, and alerting
- ✅ Cost awareness
- ✅ Automation tool (SLO Reporter)

## 🤝 Contributing

This is a take-home project. For production use, consider:

- Implementing NetworkPolicies
- Adding TLS/SSL certificates
- Setting up backup procedures
- Implementing disaster recovery
- Adding more comprehensive testing
- Setting up cost monitoring tools (Kubecost)

## 📄 License

This project is part of a technical assessment.

## 🔗 Quick Links

- [Infrastructure Setup](infra/terraform/README.md)
- [ArgoCD Setup](argocd/README.md)
- [Observability Guide](OBSERVABILITY.md)
- [Security Guide](SECURITY.md)
- [SLOs and Alerts](SLO_SLI_ALERTS.md)
- [Cost Analysis](COST.md)
- [Infrastructure CI/CD](.github/workflows/CICD_INFRA.md)
- [Application CI/CD](.github/workflows/CICD_APP.md)
- [SLO Reporter Tool](tools/slo-reporter/README.md)
- [AI Usage Notes](AI_NOTES.md)
