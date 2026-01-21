# Install ArgoCD

```bash
kubectl create namespace argocd

kubectl apply -n argocd   -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

Wait for pods:
```bash
kubectl get pods -n argocd
```

Get admin password:
```bash
kubectl -n argocd get secret argocd-initial-admin-secret   -o jsonpath="{.data.password}" | base64 -d; echo
```

Port-forward UI:
```bash
kubectl -n argocd port-forward svc/argocd-server 8080:443
```
Open: https://localhost:8080
