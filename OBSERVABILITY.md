# Observability Stack Documentation

This document describes the complete observability implementation for the Codigo application, including metrics, logs, traces, and dashboards.

## Overview

The observability stack provides comprehensive monitoring, logging, and tracing capabilities for the Codigo application (API and Worker services). It consists of:

- **Prometheus** - Metrics collection and storage
- **Grafana** - Visualization and dashboards
- **Loki** - Log aggregation
- **Tempo** - Distributed tracing backend
- **OpenTelemetry Collector** - Trace collection and forwarding

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
├─────────────────────────────────────────────────────────────┤
│  API Service                    Worker Service               │
│  - Prometheus Metrics          - Prometheus Metrics          │
│  - Structured Logs (Zap)      - Structured Logs (Zap)      │
│  - OpenTelemetry Traces        - OpenTelemetry Traces       │
│  - /metrics endpoint           - /metrics endpoint          │
└──────────────┬──────────────────────────┬──────────────────┘
               │                          │
               ▼                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Collection & Storage Layer                      │
├─────────────────────────────────────────────────────────────┤
│  Prometheus          Loki            Tempo                  │
│  - Scrapes metrics   - Collects logs  - Stores traces       │
│  - 30d retention     - 7d retention   - 48h retention       │
│  - ServiceMonitors   - Promtail       - OTLP receivers      │
└──────────────┬──────────────┬──────────────┬────────────────┘
               │              │              │
               └──────────────┴──────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Visualization Layer                       │
├─────────────────────────────────────────────────────────────┤
│                        Grafana                               │
│  - Prometheus datasource (metrics)                          │
│  - Loki datasource (logs)                                    │
│  - Tempo datasource (traces)                                 │
│  - Pre-built dashboards                                      │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Steps

### Step 1: Observability Stack in Kubernetes

The observability stack is deployed as a Helm chart located in `k8s/observability/`.

#### Components

**Prometheus (kube-prometheus-stack)**
- Version: 61.7.0
- Scrapes metrics from all namespaces via ServiceMonitors
- Retention: 30 days
- Storage: 50Gi persistent volume
- Resources: 2Gi memory, 1 CPU (requests)

**Grafana**
- Pre-configured datasources: Prometheus, Loki, Tempo
- Pre-built dashboards: Golden Signals, Dependencies
- Persistence: 10Gi
- Default credentials: admin/admin (change in production)
- Access: ClusterIP service on port 80

**Loki**
- Version: 6.16.0
- Mode: SingleBinary
- Retention: 7 days (168 hours)
- Storage: 20Gi persistent volume
- Promtail: Automatically collects logs from all Kubernetes pods

**Tempo**
- Version: 1.10.0
- Retention: 48 hours
- OTLP receivers:
  - HTTP: port 4318
  - gRPC: port 4317
- Storage: Local filesystem

**OpenTelemetry Collector**
- Version: 0.109.0
- Mode: Deployment
- Receives traces via OTLP (HTTP/gRPC)
- Forwards to Tempo
- Batch processing enabled

#### Configuration Files

- `k8s/observability/Chart.yaml` - Helm chart dependencies
- `k8s/observability/values.yaml` - Stack configuration
- `k8s/observability/templates/namespace.yaml` - Namespace creation
- `k8s/observability/templates/dashboards-configmap.yaml` - Grafana dashboards

### Step 2: Code Instrumentation

Both API and Worker services have been instrumented with:

#### Metrics (Prometheus)

**API Metrics:**
- `http_requests_total` - Total HTTP requests (labels: service, route, method, code)
- `http_request_duration_seconds` - Request latency histogram (labels: service, route, method)
- `db_connections_active` - Active database connections (label: service)
- `nats_messages_published_total` - NATS messages published (labels: service, subject)

**Worker Metrics:**
- `jobs_processed_total` - Total jobs processed (labels: service, result)
- `job_processing_duration_seconds` - Job processing duration (label: service)
- `db_connections_active` - Active database connections (label: service)
- `nats_messages_received_total` - NATS messages received (labels: service, subject)

**Metrics Endpoints:**
- API: `http://codigo-api:8080/metrics`
- Worker: `http://codigo-worker:8080/metrics`

#### Logs (Structured Logging with Zap)

**Features:**
- JSON format for log aggregation
- Trace ID and Span ID included in all logs for correlation
- Log levels: Debug, Info, Warn, Error
- Contextual information: method, route, status code, duration, errors

**Example Log Entry:**
```json
{
  "level": "info",
  "ts": 1234567890.123,
  "caller": "main.go:90",
  "msg": "http request",
  "trace_id": "abc123...",
  "span_id": "def456...",
  "method": "GET",
  "route": "/v1/jobs",
  "status_code": 200,
  "duration": 0.045
}
```

#### Traces (OpenTelemetry)

**Trace Propagation:**
- API propagates trace context in HTTP headers
- API propagates trace context in NATS message headers
- Worker extracts trace context from NATS headers
- End-to-end trace correlation: API → Worker

**Span Attributes:**
- API spans: job.id, http.method, http.route, http.status_code, http.duration_ms
- Worker spans: job.id, nats.subject, job.status, job.duration_ms

**Trace Export:**
- Endpoint: Configured via `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable
- Default: `http://otel-collector.observability:4318`
- Protocol: OTLP HTTP

### Step 3: Kubernetes Configuration

#### ServiceMonitors

ServiceMonitors enable Prometheus to automatically discover and scrape metrics:

**API ServiceMonitor** (`k8s/apps/codigo/templates/servicemonitor-api.yaml`):
```yaml
- Scrapes: codigo-api service
- Endpoint: /metrics on port http
- Interval: 30 seconds
- Timeout: 10 seconds
```

**Worker ServiceMonitor** (`k8s/apps/codigo/templates/servicemonitor-worker.yaml`):
```yaml
- Scrapes: codigo-worker service
- Endpoint: /metrics on port metrics
- Interval: 30 seconds
- Timeout: 10 seconds
```

#### Services

**API Service** (`k8s/apps/codigo/templates/api-service.yaml`):
- Port `http`: 80 → 8080 (application)
- Port `metrics`: 8080 → 8080 (metrics endpoint)

**Worker Service** (`k8s/apps/codigo/templates/worker-service.yaml`):
- Port `metrics`: 8080 → 8080 (metrics endpoint)

### Step 4: ArgoCD Integration

The observability stack is integrated with ArgoCD using the App-of-Apps pattern:

**Application Definition** (`argocd/apps/app-of-apps/templates/apps.yaml`):
```yaml
name: codigo-observability
source:
  path: k8s/observability
  helm:
    valueFiles:
      - values.yaml
syncPolicy:
  automated:
    prune: true
    selfHeal: true
```

**Features:**
- Automatic deployment when changes are pushed to Git
- Self-healing: reverts manual changes
- Prune: removes deleted resources
- Namespace creation: automatic

## Accessing the Stack

### Grafana

**Port Forward:**
```bash
kubectl port-forward svc/kube-prometheus-stack-grafana -n observability 3000:80
```

**Access:**
- URL: http://localhost:3000
- Username: `admin`
- Password: `admin` (default, change in production)

**Dashboards:**
- **Golden Signals**: Latency (p50, p95, p99), Traffic, Error Rate, Saturation
- **Dependencies**: PostgreSQL metrics, NATS metrics

### Prometheus

**Port Forward:**
```bash
kubectl port-forward svc/kube-prometheus-stack-prometheus -n observability 9090:9090
```

**Access:**
- URL: http://localhost:9090
- Query examples:
  - `rate(http_requests_total[5m])`
  - `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`

### Loki

**Port Forward:**
```bash
kubectl port-forward svc/loki -n observability 3100:3100
```

**Access:**
- URL: http://localhost:3100
- LogQL queries:
  - `{namespace="codigo"}`
  - `{app="codigo-api"} |= "error"`

### Tempo

**Port Forward:**
```bash
kubectl port-forward svc/tempo -n observability 3200:3200
```

**Access:**
- URL: http://localhost:3200
- Query traces by trace ID or service name

## Dashboards

### Golden Signals Dashboard

**Location:** Grafana → Dashboards → Codigo → Golden Signals

**Panels:**
1. **Latency (p50, p95, p99)**
   - Query: `histogram_quantile(0.50/0.95/0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, service))`
   - Shows request latency percentiles

2. **Traffic (Requests per Second)**
   - Query: `sum(rate(http_requests_total[5m])) by (service)`
   - Shows request rate per service

3. **Error Rate**
   - Query: `sum(rate(http_requests_total{status=~"5.."}[5m])) by (service) / sum(rate(http_requests_total[5m])) by (service)`
   - Shows percentage of 5xx errors

4. **Saturation (CPU & Memory)**
   - Query: `avg(container_cpu_usage_seconds_total{namespace="codigo"}) by (pod) * 100`
   - Query: `avg(container_memory_usage_bytes{namespace="codigo"}) by (pod) / 1024 / 1024`
   - Shows resource utilization

### Dependencies Dashboard

**Location:** Grafana → Dashboards → Codigo → Dependencies

**Panels:**
1. **PostgreSQL - Connections & Transactions**
   - Active connections, commits, rollbacks

2. **PostgreSQL - Query Latency**
   - Query latency percentiles (p95, p99)

3. **NATS - Memory & Connections**
   - Memory usage, active connections

4. **NATS - Messages & Subscriptions**
   - Message rate, subscription count

## Log Correlation

Logs are automatically correlated with traces:

1. **In Grafana:**
   - Open a trace in Tempo
   - Click on a span
   - Click "Logs" to see related logs from Loki

2. **Trace ID in Logs:**
   - All logs include `trace_id` field
   - Search in Loki: `{namespace="codigo"} | json | trace_id="abc123..."`

3. **Span ID in Logs:**
   - All logs include `span_id` field
   - Useful for correlating specific operations

## Environment Variables

### Application Configuration

**API and Worker:**
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OpenTelemetry collector endpoint
  - Default: `http://otel-collector.observability:4318`
- `SERVICE_NAME` - Service name for metrics and traces
  - API: `codigo-api`
  - Worker: `codigo-worker`

**Set in Kubernetes:**
- `k8s/apps/codigo/templates/api-deployment.yaml`
- `k8s/apps/codigo/templates/worker-deployment.yaml`

## Troubleshooting

### Metrics Not Appearing

1. **Check ServiceMonitors:**
   ```bash
   kubectl get servicemonitors -n codigo
   kubectl describe servicemonitor codigo-api -n codigo
   ```

2. **Check Prometheus Targets:**
   - Access Prometheus UI
   - Go to Status → Targets
   - Verify API and Worker are being scraped

3. **Check Metrics Endpoint:**
   ```bash
   kubectl exec -it deployment/codigo-api -n codigo -- wget -qO- http://localhost:8080/metrics
   ```

### Logs Not Appearing in Loki

1. **Check Promtail:**
   ```bash
   kubectl get pods -n observability -l app.kubernetes.io/name=promtail
   kubectl logs -n observability -l app.kubernetes.io/name=promtail
   ```

2. **Check Loki:**
   ```bash
   kubectl get pods -n observability -l app.kubernetes.io/name=loki
   kubectl logs -n observability -l app.kubernetes.io/name=loki
   ```

3. **Verify Log Labels:**
   - Check pod labels match Promtail configuration
   - Verify namespace is correct

### Traces Not Appearing in Tempo

1. **Check OpenTelemetry Collector:**
   ```bash
   kubectl get pods -n observability -l app.kubernetes.io/name=opentelemetry-collector
   kubectl logs -n observability -l app.kubernetes.io/name=opentelemetry-collector
   ```

2. **Check Tempo:**
   ```bash
   kubectl get pods -n observability -l app.kubernetes.io/name=tempo
   kubectl logs -n observability -l app.kubernetes.io/name=tempo
   ```

3. **Verify OTLP Endpoint:**
   ```bash
   kubectl get deployment codigo-api -n codigo -o yaml | grep OTEL_EXPORTER_OTLP_ENDPOINT
   ```

### Grafana Dashboards Not Loading

1. **Check Dashboard ConfigMap:**
   ```bash
   kubectl get configmap codigo-dashboards -n observability
   kubectl describe configmap codigo-dashboards -n observability
   ```

2. **Check Grafana Logs:**
   ```bash
   kubectl logs -n observability -l app.kubernetes.io/name=grafana
   ```

3. **Verify Dashboard Provisioning:**
   - Check Grafana UI → Configuration → Dashboards
   - Verify "Codigo" folder exists

### High Resource Usage

1. **Check Prometheus Retention:**
   - Current: 30 days
   - Adjust in `k8s/observability/values.yaml`:
     ```yaml
     prometheus:
       prometheusSpec:
         retention: 7d  # Reduce if needed
     ```

2. **Check Loki Retention:**
   - Current: 7 days
   - Adjust in `k8s/observability/values.yaml`:
     ```yaml
     loki:
       config:
         limits_config:
           retention_period: 72h  # Reduce if needed
     ```

3. **Check Tempo Retention:**
   - Current: 48 hours
   - Adjust in `k8s/observability/values.yaml`:
     ```yaml
     tempo:
       tempo:
         retention: 24h  # Reduce if needed
     ```

## Best Practices

### Metrics

1. **Use Appropriate Buckets:**
   - HTTP latency: Custom buckets for better resolution
   - Job processing: Custom buckets based on expected duration

2. **Label Cardinality:**
   - Avoid high-cardinality labels (e.g., user IDs)
   - Use labels for dimensions that are useful for aggregation

3. **Metric Naming:**
   - Follow Prometheus naming conventions
   - Use `_total` suffix for counters
   - Use `_seconds` suffix for duration

### Logs

1. **Structured Logging:**
   - Always use structured logs (JSON)
   - Include trace_id and span_id for correlation
   - Use appropriate log levels

2. **Log Volume:**
   - Avoid logging sensitive information
   - Use log levels appropriately (don't log everything as INFO)

### Traces

1. **Span Attributes:**
   - Add meaningful attributes to spans
   - Include business context (e.g., job.id, user.id)
   - Keep attribute values reasonable in size

2. **Trace Sampling:**
   - Consider sampling for high-traffic services
   - Currently: 100% sampling (adjust if needed)

## Production Considerations

### Security

1. **Grafana Credentials:**
   - Change default admin password
   - Use secrets for sensitive configuration
   - Enable RBAC

2. **Network Policies:**
   - Restrict access to observability namespace
   - Only allow necessary ingress/egress

3. **TLS:**
   - Enable TLS for Grafana
   - Use certificates for Prometheus scraping

### Scalability

1. **Prometheus:**
   - Consider Prometheus federation for multiple clusters
   - Use Thanos for long-term storage

2. **Loki:**
   - Consider Loki cluster mode for high volume
   - Use object storage backend

3. **Tempo:**
   - Consider Tempo cluster mode for high volume
   - Use object storage backend

### High Availability

1. **Replicas:**
   - Increase replicas for critical components
   - Use PodDisruptionBudgets

2. **Storage:**
   - Use reliable storage classes
   - Consider backup strategies

## Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Loki Documentation](https://grafana.com/docs/loki/)
- [Tempo Documentation](https://grafana.com/docs/tempo/)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)

## Summary

The observability stack provides:

✅ **Metrics** - Prometheus scraping API and Worker metrics  
✅ **Logs** - Loki collecting structured logs from all pods  
✅ **Traces** - Tempo storing distributed traces  
✅ **Dashboards** - Grafana with Golden Signals and Dependencies  
✅ **GitOps** - ArgoCD managing automatic deployments  
✅ **Correlation** - Logs, metrics, and traces linked via trace IDs  

All components are production-ready with persistence, resource limits, and automated reconciliation.

