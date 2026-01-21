# Security Baseline

This document describes the security measures implemented for the Codigo application.

## Security Requirements

✅ **No plaintext secrets in Git**  
✅ **RBAC and namespace separation**  
✅ **Pod security hardening (non-root, minimal privileges)**

## 1. Secrets Management

### Current Implementation

**Secrets are NOT stored in Git.** All sensitive data is managed via Kubernetes Secrets.

### PostgreSQL Password

The PostgreSQL password is stored in a Kubernetes Secret (`postgres-secret`). 

**⚠️ IMPORTANT:** The secret template in `k8s/apps/codigo/templates/postgres.yaml` contains a placeholder value `CHANGE_ME_REPLACE_WITH_SECRET`. This MUST be replaced before deployment.

### Creating Secrets

**Option 1: Manual Secret Creation**
```bash
# Generate a secure password
PASSWORD=$(openssl rand -base64 32)

# Create the secret
kubectl create secret generic postgres-secret \
  --from-literal=POSTGRES_PASSWORD=$PASSWORD \
  -n codigo
```

**Option 2: Using Sealed Secrets (Recommended for GitOps)**
```bash
# Install Sealed Secrets controller
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.0/controller.yaml

# Create sealed secret
kubectl create secret generic postgres-secret \
  --from-literal=POSTGRES_PASSWORD=$PASSWORD \
  --dry-run=client -o yaml | \
  kubeseal -o yaml > sealed-secret.yaml

# Commit sealed-secret.yaml to Git (safe to commit)
```

**Option 3: Using External Secrets Operator**
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: postgres-secret
  namespace: codigo
spec:
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: postgres-secret
  data:
    - secretKey: POSTGRES_PASSWORD
      remoteRef:
        key: postgres/password
```

### Grafana Admin Password

The Grafana admin password is configured in `k8s/observability/values.yaml`. 

**⚠️ IMPORTANT:** Change the default password before production deployment:

```yaml
grafana:
  adminPassword: "CHANGE_ME_USE_SECURE_PASSWORD"
```

Or use a secret:
```yaml
grafana:
  adminUser: admin
  adminPassword: $__file{secrets/grafana-password.txt}
```

### Application Defaults

The application code has default values for development only:
- `getenv("POSTGRES_PASSWORD", "codigo")` - Development default
- These defaults should NEVER be used in production
- Always set environment variables via Kubernetes Secrets

## 2. RBAC and Namespace Separation

### Namespace Isolation

All application components are deployed in the `codigo` namespace, providing isolation from other workloads:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: codigo
```

### Service Accounts

Dedicated ServiceAccounts are created for each component:

**API ServiceAccount:**
- Name: `codigo-api`
- Namespace: `codigo`
- Used by: `codigo-api` deployment

**Worker ServiceAccount:**
- Name: `codigo-worker`
- Namespace: `codigo`
- Used by: `codigo-worker` deployment

### RBAC Configuration

Currently, ServiceAccounts are created with minimal permissions. Additional RBAC can be added as needed:

**Example: If API needs to read ConfigMaps:**
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: codigo-api-reader
  namespace: codigo
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: codigo-api-reader-binding
  namespace: codigo
subjects:
  - kind: ServiceAccount
    name: codigo-api
    namespace: codigo
roleRef:
  kind: Role
  name: codigo-api-reader
  apiGroup: rbac.authorization.k8s.io
```

### Network Policies (Optional Enhancement)

For additional network isolation, consider implementing NetworkPolicies:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: codigo-api-policy
  namespace: codigo
spec:
  podSelector:
    matchLabels:
      app: codigo-api
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: observability
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: postgres
      ports:
        - protocol: TCP
          port: 5432
    - to:
        - podSelector:
            matchLabels:
              app: nats
      ports:
        - protocol: TCP
          port: 4222
```

## 3. Pod Security Hardening

### Security Context Configuration

All pods implement the following security hardening:

#### Pod-Level Security Context

```yaml
securityContext:
  runAsNonRoot: true        # Prevent running as root
  runAsUser: 65534          # Run as 'nobody' user (non-root)
  fsGroup: 65534            # Filesystem group
  seccompProfile:
    type: RuntimeDefault     # Use default seccomp profile
```

#### Container-Level Security Context

```yaml
securityContext:
  allowPrivilegeEscalation: false  # Prevent privilege escalation
  readOnlyRootFilesystem: true     # Read-only root filesystem
  runAsNonRoot: true               # Enforce non-root
  runAsUser: 65534                  # Run as 'nobody' user
  capabilities:
    drop:
      - ALL                         # Drop all Linux capabilities
```

### Component-Specific Configurations

**API and Worker:**
- User: `65534` (nobody)
- Read-only root filesystem
- Writable volumes: `/tmp`, `/var/run` (emptyDir)

**PostgreSQL:**
- User: `999` (postgres user in official image)
- Read-only root filesystem (PostgreSQL stores data in mounted volumes)

**NATS:**
- User: `1000` (nats user in official image)
- Read-only root filesystem

### Writable Volumes

Since we use `readOnlyRootFilesystem: true`, writable volumes are mounted for temporary files:

```yaml
volumeMounts:
  - name: tmp
    mountPath: /tmp
  - name: var-run
    mountPath: /var/run
volumes:
  - name: tmp
    emptyDir: {}
  - name: var-run
    emptyDir: {}
```

## Security Checklist

### Pre-Deployment

- [ ] Replace `CHANGE_ME_REPLACE_WITH_SECRET` in postgres-secret.yaml
- [ ] Change Grafana admin password from default
- [ ] Verify all secrets are created before deployment
- [ ] Review and adjust resource limits
- [ ] Verify ServiceAccounts are created

### Post-Deployment

- [ ] Verify pods are running as non-root:
  ```bash
  kubectl get pods -n codigo -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.securityContext.runAsUser}{"\n"}{end}'
  ```
- [ ] Verify read-only root filesystem:
  ```bash
  kubectl exec -it deployment/codigo-api -n codigo -- touch /test
  # Should fail with "read-only file system"
  ```
- [ ] Verify no privilege escalation:
  ```bash
  kubectl exec -it deployment/codigo-api -n codigo -- sh -c 'id'
  # Should show uid=65534(nobody)
  ```

## Additional Security Recommendations

### 1. Image Security

- Use distroless or minimal base images (already implemented)
- Regularly scan images for vulnerabilities
- Use image signing and verification

### 2. Secrets Rotation

- Implement regular secret rotation
- Use external secret management (Vault, AWS Secrets Manager, etc.)
- Monitor secret access

### 3. Network Security

- Implement NetworkPolicies (see example above)
- Use TLS for all inter-service communication
- Restrict egress traffic

### 4. Monitoring and Auditing

- Enable Kubernetes audit logging
- Monitor for security events
- Set up alerts for suspicious activity

### 5. Compliance

- Follow CIS Kubernetes Benchmark
- Implement Pod Security Standards
- Regular security assessments

## Pod Security Standards

The current configuration aligns with **Restricted** Pod Security Standard:

- ✅ `runAsNonRoot: true`
- ✅ `readOnlyRootFilesystem: true`
- ✅ `allowPrivilegeEscalation: false`
- ✅ `capabilities.drop: ["ALL"]`
- ✅ `seccompProfile.type: RuntimeDefault`

To enforce at namespace level:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: codigo
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

## Troubleshooting

### Pod Fails to Start: "container has runAsNonRoot and image will run as root"

**Solution:** Ensure the container image runs as non-root user, or adjust the securityContext to match the image's user.

### Pod Fails: "read-only file system"

**Solution:** Ensure all writable paths are mounted as volumes (e.g., `/tmp`, `/var/run`).

### Application Needs Additional Permissions

**Solution:** Create appropriate RBAC resources (Role, RoleBinding) and assign to the ServiceAccount.

## Summary

✅ **Secrets:** No plaintext secrets in Git - all managed via Kubernetes Secrets  
✅ **RBAC:** ServiceAccounts created, namespace isolation implemented  
✅ **Pod Security:** Non-root, read-only filesystem, minimal capabilities, seccomp enabled  

All security requirements are met. Additional enhancements (NetworkPolicies, external secret management) can be added as needed.

