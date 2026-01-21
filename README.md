# Codigo SRE Take-Home (Starter)

This repo provides a baseline skeleton for the Codigo SRE take-home project:
- GKE + GCP resources via Terraform
- GitOps via ArgoCD (app-of-apps)
- Observability stack: Prometheus/Grafana, Loki, Tempo, OpenTelemetry
- Sample Go API + Worker using Postgres + NATS

## Quick Start (intended flow)
1) Provision infra (GKE + Artifact Registry):
```bash
cd infra/terraform
terraform init
terraform apply
```

2) Install ArgoCD into the cluster:
See: `argocd/bootstrap/install-argocd.md`

3) Bootstrap app-of-apps:
```bash
kubectl apply -n argocd -f argocd/bootstrap/root-app.yaml
```

4) Verify:
- ArgoCD sync: all apps Healthy
- API reachable (Ingress or port-forward)
- Grafana dashboard shows traffic/latency/errors
- Traces visible in Tempo
- Logs visible in Loki

## Candidate Tasks
You must extend and improve this baseline:
- Harden security (RBAC, NetworkPolicies, secrets, pod security)
- Improve CI/CD
- Define SLIs/SLOs and meaningful alerts
- Add runbooks
- Add cost visibility & optimizations
- Add an automation tool (Go or TypeScript)

## Notes
You may use AI tools, but include an `AI_NOTES.md` describing what you used them for.
