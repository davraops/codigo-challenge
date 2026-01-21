# ArgoCD GitOps Setup

This directory contains the ArgoCD configuration for GitOps deployment of the Codigo application and observability stack.

## Architecture

The setup uses the **App-of-Apps** pattern:
- **Root Application** (`bootstrap/root-app.yaml`): Bootstraps the app-of-apps pattern
- **App-of-Apps** (`apps/app-of-apps/`): Helm chart that manages multiple ArgoCD Applications
  - `codigo-app`: Deploys the main application (API + Worker + dependencies)
  - `codigo-observability`: Deploys the observability stack (Prometheus, Grafana, Loki, Tempo, OpenTelemetry)

## Features

✅ **Automated Reconciliation**: Both applications have automated sync enabled  
✅ **Self-Heal**: Applications automatically revert manual changes  
✅ **Prune**: Orphaned resources are automatically removed  
✅ **GitOps**: All deployments are managed via Git

## Setup Instructions

### 1. Install ArgoCD

Follow the instructions in `bootstrap/install-argocd.md`:

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

Wait for ArgoCD to be ready:
```bash
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server -n argocd --timeout=300s
```

### 2. Configure Repository URL

Before bootstrapping, update the repository URL in `bootstrap/root-app.yaml`:

```yaml
source:
  repoURL: https://github.com/your-org/codigo-challenge  # Update this
  targetRevision: HEAD
```

Or set it via environment variable when applying:
```bash
REPO_URL="https://github.com/your-org/codigo-challenge"
sed "s|REPO_URL|${REPO_URL}|g" argocd/bootstrap/root-app.yaml | kubectl apply -f -
```

### 3. Bootstrap App-of-Apps

Apply the root application:

```bash
kubectl apply -n argocd -f argocd/bootstrap/root-app.yaml
```

This will create the root application, which in turn creates:
- `codigo-app` application (deploys `k8s/apps/codigo`)
- `codigo-observability` application (deploys `k8s/observability`)

### 4. Verify Deployment

Check ArgoCD applications:
```bash
kubectl get applications -n argocd
```

Expected output:
```
NAME                  SYNC STATUS   HEALTH STATUS
codigo-app            Synced        Healthy
codigo-observability  Synced        Healthy
codigo-root           Synced        Healthy
```

Access ArgoCD UI:
```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Open https://localhost:8080
# Username: admin
# Password: $(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)
```

## Sync Policies

Both applications are configured with automated sync policies:

```yaml
syncPolicy:
  automated:
    prune: true      # Remove resources deleted from Git
    selfHeal: true   # Automatically revert manual changes
```

This means:
- **Automatic Sync**: Changes in Git are automatically deployed
- **Self-Heal**: Manual changes to resources are automatically reverted
- **Prune**: Resources deleted from Git are automatically removed from cluster

## Application Details

### codigo-app

- **Path**: `k8s/apps/codigo`
- **Namespace**: `codigo`
- **Components**:
  - API deployment and service
  - Worker deployment
  - PostgreSQL database
  - NATS message queue
  - PrometheusRule for alerts

### codigo-observability

- **Path**: `k8s/observability`
- **Namespace**: `observability`
- **Components**:
  - Prometheus (via kube-prometheus-stack)
  - Grafana with Loki and Tempo data sources
  - Loki for log aggregation
  - Tempo for distributed tracing
  - OpenTelemetry Collector for trace collection

## Troubleshooting

### Applications not syncing

1. Check application status:
   ```bash
   kubectl describe application codigo-app -n argocd
   ```

2. Check repository access:
   - Ensure ArgoCD can access the Git repository
   - For private repos, configure repository credentials in ArgoCD

3. Force sync via UI or CLI:
   ```bash
   argocd app sync codigo-app
   ```

### Self-heal not working

- Verify sync policy is set correctly:
  ```bash
  kubectl get application codigo-app -n argocd -o yaml | grep -A 3 syncPolicy
  ```

### Resources not pruning

- Ensure `prune: true` is set in sync policy
- Manually delete resources if needed:
  ```bash
  argocd app delete codigo-app
  ```

## GitOps Workflow

1. **Make changes** to Kubernetes manifests in `k8s/` directory
2. **Commit and push** to Git repository
3. **ArgoCD detects changes** (within sync window)
4. **Automatic sync** deploys changes to cluster
5. **Self-heal** reverts any manual changes

## Next Steps

- Configure repository credentials for private repos
- Set up webhooks for faster sync (optional)
- Configure RBAC for ArgoCD access
- Add additional applications as needed

