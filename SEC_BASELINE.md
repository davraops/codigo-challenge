# Security Baseline Summary

This document provides a concise summary of the security baseline implementation for the Codigo application.

## Requirements Met

✅ **No plaintext secrets committed in Git**  
✅ **Reasonable RBAC and namespace separation**  
✅ **Pod security hardening (non-root, minimal privileges)**

## 1. Secrets Management

### Status: ✅ COMPLIANT

**Changes:**
- Removed hardcoded password from `values.yaml`
- Removed default password fallback from application code
- PostgreSQL secret uses placeholder: `CHANGE_ME_REPLACE_WITH_SECRET`
- All secrets must be created via Kubernetes Secrets or external secret management

**Before:**
```yaml
# values.yaml
postgres:
  password: codigo  # ❌ Hardcoded in Git
```

```go
// main.go
pass := getenv("POSTGRES_PASSWORD", "codigo")  // ❌ Default fallback
```

**After:**
```yaml
# values.yaml
postgres:
  # password: Set via secret - DO NOT commit passwords here
```

```go
// main.go
pass := os.Getenv("POSTGRES_PASSWORD")
if pass == "" {
    panic("POSTGRES_PASSWORD environment variable is required")  // ✅ Fails if not set
}
```

**Secret Creation:**
```bash
kubectl create secret generic postgres-secret \
  --from-literal=POSTGRES_PASSWORD=<secure-password> \
  -n codigo
```

## 2. RBAC and Namespace Separation

### Status: ✅ COMPLIANT

**Implementation:**
- **Namespace Isolation:** All components deployed in `codigo` namespace
- **ServiceAccounts:** Dedicated ServiceAccounts for each component
  - `codigo-api` ServiceAccount for API pods
  - `codigo-worker` ServiceAccount for Worker pods
- **RBAC Ready:** ServiceAccounts created, ready for Role/RoleBinding if needed

**Files Created:**
- `k8s/apps/codigo/templates/serviceaccount-api.yaml`
- `k8s/apps/codigo/templates/serviceaccount-worker.yaml`

**Deployment Configuration:**
```yaml
spec:
  serviceAccountName: codigo-api  # ✅ Dedicated ServiceAccount
```

## 3. Pod Security Hardening

### Status: ✅ COMPLIANT

**Security Context Applied to All Pods:**

#### Pod-Level
```yaml
securityContext:
  runAsNonRoot: true           # ✅ Non-root user
  runAsUser: 65534             # ✅ nobody user (API/Worker)
  fsGroup: 65534               # ✅ Filesystem group
  seccompProfile:
    type: RuntimeDefault        # ✅ Seccomp enabled
```

#### Container-Level
```yaml
securityContext:
  allowPrivilegeEscalation: false  # ✅ No privilege escalation
  readOnlyRootFilesystem: true     # ✅ Read-only root
  runAsNonRoot: true              # ✅ Enforce non-root
  runAsUser: 65534                 # ✅ nobody user
  capabilities:
    drop:
      - ALL                        # ✅ Drop all capabilities
```

**Component-Specific Users:**
- **API/Worker:** User `65534` (nobody)
- **PostgreSQL:** User `999` (postgres)
- **NATS:** User `1000` (nats)

**Writable Volumes:**
- `/tmp` and `/var/run` mounted as emptyDir volumes (required for read-only root)

## Security Checklist

### Pre-Deployment
- [x] Secrets removed from Git
- [x] ServiceAccounts created
- [x] SecurityContext applied to all pods
- [x] Non-root users configured
- [x] Read-only root filesystem enabled
- [x] Capabilities dropped
- [ ] **TODO:** Create PostgreSQL secret before deployment
- [ ] **TODO:** Change Grafana admin password

### Post-Deployment Verification

**Verify Non-Root:**
```bash
kubectl get pods -n codigo -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.securityContext.runAsUser}{"\n"}{end}'
# Expected: All pods show non-root UID (65534, 999, or 1000)
```

**Verify Read-Only Root:**
```bash
kubectl exec -it deployment/codigo-api -n codigo -- touch /test
# Expected: Error "read-only file system"
```

**Verify ServiceAccounts:**
```bash
kubectl get pods -n codigo -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.serviceAccountName}{"\n"}{end}'
# Expected: codigo-api and codigo-worker show their ServiceAccounts
```

## Files Modified

### Security Hardening
- `k8s/apps/codigo/templates/api-deployment.yaml` - Added securityContext, ServiceAccount
- `k8s/apps/codigo/templates/worker-deployment.yaml` - Added securityContext, ServiceAccount
- `k8s/apps/codigo/templates/postgres.yaml` - Added securityContext, secret placeholder
- `k8s/apps/codigo/templates/nats.yaml` - Added securityContext

### Secrets Management
- `k8s/apps/codigo/values.yaml` - Removed hardcoded password
- `k8s/apps/codigo/templates/postgres.yaml` - Secret placeholder
- `app/api/main.go` - Removed default password fallback
- `app/worker/main.go` - Removed default password fallback

### RBAC
- `k8s/apps/codigo/templates/serviceaccount-api.yaml` - Created
- `k8s/apps/codigo/templates/serviceaccount-worker.yaml` - Created

## Pod Security Standards Compliance

The implementation aligns with **Kubernetes Pod Security Standard: Restricted**

| Requirement | Status | Implementation |
|------------|--------|----------------|
| runAsNonRoot | ✅ | `runAsNonRoot: true` |
| readOnlyRootFilesystem | ✅ | `readOnlyRootFilesystem: true` |
| allowPrivilegeEscalation | ✅ | `allowPrivilegeEscalation: false` |
| capabilities.drop | ✅ | `capabilities.drop: ["ALL"]` |
| seccompProfile | ✅ | `seccompProfile.type: RuntimeDefault` |

## Security Posture

### Current State
- ✅ **Secrets:** No plaintext secrets in Git
- ✅ **RBAC:** ServiceAccounts with namespace isolation
- ✅ **Pod Security:** Hardened with Restricted policy compliance
- ✅ **Network:** Namespace isolation (NetworkPolicies optional enhancement)

### Optional Enhancements
- NetworkPolicies for network-level isolation
- External Secrets Operator for secret management
- Pod Security Standards enforcement at namespace level
- Image scanning and signing
- TLS for inter-service communication

## Quick Reference

### Create PostgreSQL Secret
```bash
PASSWORD=$(openssl rand -base64 32)
kubectl create secret generic postgres-secret \
  --from-literal=POSTGRES_PASSWORD=$PASSWORD \
  -n codigo
```

### Verify Security Context
```bash
# Check all pods run as non-root
kubectl get pods -n codigo -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.securityContext.runAsUser}{"\n"}{end}'
```

### Check ServiceAccounts
```bash
kubectl get serviceaccounts -n codigo
kubectl get pods -n codigo -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.serviceAccountName}{"\n"}{end}'
```

## Summary

All security baseline requirements are met:

1. ✅ **No plaintext secrets** - Removed from Git, managed via Kubernetes Secrets
2. ✅ **RBAC and namespace separation** - ServiceAccounts created, namespace isolation
3. ✅ **Pod security hardening** - Non-root, read-only filesystem, minimal privileges, seccomp

The implementation follows Kubernetes security best practices and aligns with the Restricted Pod Security Standard.

