# SLIs, SLOs, and Alerting

This document defines Service Level Indicators (SLIs), Service Level Objectives (SLOs), alerting rules, and runbooks for the Codigo application.

## SLIs (Service Level Indicators)

### 1. Availability
**Definition:** Percentage of successful HTTP requests (non-5xx responses) over total requests.

**Measurement:**
```promql
sum(rate(http_requests_total{service=~"codigo-api", code!~"5.."}[5m])) 
/ 
sum(rate(http_requests_total{service=~"codigo-api"}[5m]))
```

**Window:** Rolling 30-day window

### 2. Latency (p95)
**Definition:** 95th percentile of HTTP request duration.

**Measurement:**
```promql
histogram_quantile(0.95, 
  sum(rate(http_request_duration_seconds_bucket{service=~"codigo-api"}[5m])) 
  by (le, service)
)
```

**Window:** Rolling 30-day window

### 3. Job Success Rate
**Definition:** Percentage of successfully processed jobs (status='done') over total jobs processed.

**Measurement:**
```promql
sum(rate(jobs_processed_total{service="codigo-worker", result="ok"}[5m])) 
/ 
sum(rate(jobs_processed_total{service="codigo-worker"}[5m]))
```

**Window:** Rolling 30-day window

## SLOs (Service Level Objectives)

### SLO 1: API Availability
**Target:** 99.9% availability (99.9% of requests succeed)

**Error Budget:** 0.1% (43.2 minutes of downtime per month)

**Measurement Period:** 30 days

**SLI:** Availability (SLI #1)

### SLO 2: API Latency
**Target:** 95% of requests complete within 500ms (p95 ≤ 500ms)

**Error Budget:** 5% of requests can exceed 500ms

**Measurement Period:** 30 days

**SLI:** Latency p95 (SLI #2)

## Alerting Rules

### Alert 1: High Error Rate (Availability SLO at Risk)

```yaml
groups:
  - name: codigo_slo_alerts
    interval: 1m
    rules:
      - alert: HighErrorRate
        expr: |
          (
            sum(rate(http_requests_total{service=~"codigo-api", code=~"5.."}[5m])) 
            / 
            sum(rate(http_requests_total{service=~"codigo-api"}[5m]))
          ) > 0.01
        for: 5m
        labels:
          severity: warning
          slo: availability
        annotations:
          summary: "High error rate detected - Availability SLO at risk"
          description: |
            Error rate is {{ $value | humanizePercentage }} (threshold: 1%).
            Current availability: {{ (1 - $value) | humanizePercentage }}.
            SLO target: 99.9%.
            This alert fires when error rate exceeds 1% for 5 minutes.
```

**Threshold:** Error rate > 1% for 5 minutes  
**Severity:** Warning  
**SLO Impact:** Availability SLO at risk

### Alert 2: High Latency (Latency SLO at Risk)

```yaml
      - alert: HighLatency
        expr: |
          histogram_quantile(0.95,
            sum(rate(http_request_duration_seconds_bucket{service=~"codigo-api"}[5m]))
            by (le, service)
          ) > 0.5
        for: 5m
        labels:
          severity: warning
          slo: latency
        annotations:
          summary: "High latency detected - Latency SLO at risk"
          description: |
            p95 latency is {{ $value | humanizeDuration }} (threshold: 500ms).
            SLO target: p95 ≤ 500ms.
            This alert fires when p95 latency exceeds 500ms for 5 minutes.
```

**Threshold:** p95 latency > 500ms for 5 minutes  
**Severity:** Warning  
**SLO Impact:** Latency SLO at risk

### Alert 3: Critical Error Rate (Availability SLO Breached)

```yaml
      - alert: CriticalErrorRate
        expr: |
          (
            sum(rate(http_requests_total{service=~"codigo-api", code=~"5.."}[5m])) 
            / 
            sum(rate(http_requests_total{service=~"codigo-api"}[5m]))
          ) > 0.05
        for: 2m
        labels:
          severity: critical
          slo: availability
        annotations:
          summary: "Critical error rate - Availability SLO breached"
          description: |
            Error rate is {{ $value | humanizePercentage }} (threshold: 5%).
            Current availability: {{ (1 - $value) | humanizePercentage }}.
            SLO target: 99.9%.
            This alert fires when error rate exceeds 5% for 2 minutes.
```

**Threshold:** Error rate > 5% for 2 minutes  
**Severity:** Critical  
**SLO Impact:** Availability SLO breached

### Alert 4: Job Processing Failure

```yaml
      - alert: JobProcessingFailure
        expr: |
          (
            sum(rate(jobs_processed_total{service="codigo-worker", result="error"}[5m])) 
            / 
            sum(rate(jobs_processed_total{service="codigo-worker"}[5m]))
          ) > 0.1
        for: 5m
        labels:
          severity: warning
          component: worker
        annotations:
          summary: "High job processing failure rate"
          description: |
            Job failure rate is {{ $value | humanizePercentage }} (threshold: 10%).
            This alert fires when job failure rate exceeds 10% for 5 minutes.
```

**Threshold:** Job failure rate > 10% for 5 minutes  
**Severity:** Warning  
**Component:** Worker

## Runbooks

### Runbook 1: High Error Rate Alert

**Alert:** `HighErrorRate`  
**Severity:** Warning  
**SLO:** Availability

#### Symptoms
- Error rate > 1% for 5 minutes
- Availability below 99.9% target
- 5xx HTTP status codes increasing

#### Investigation Steps

1. **Check Current Error Rate:**
   ```bash
   # Query Prometheus
   sum(rate(http_requests_total{service=~"codigo-api", code=~"5.."}[5m])) 
   / 
   sum(rate(http_requests_total{service=~"codigo-api"}[5m]))
   ```

2. **Identify Error Types:**
   ```bash
   # Check error codes breakdown
   sum(rate(http_requests_total{service=~"codigo-api", code=~"5.."}[5m])) by (code)
   ```

3. **Check Application Logs:**
   ```bash
   # Query Loki for errors
   {namespace="codigo", app="codigo-api"} |= "error" | json
   ```

4. **Check Dependencies:**
   ```bash
   # Check database connectivity
   kubectl exec -it deployment/codigo-api -n codigo -- curl http://localhost:8080/readyz
   
   # Check NATS connectivity
   kubectl get pods -n codigo -l app=nats
   ```

5. **Check Resource Usage:**
   ```bash
   # Check CPU and memory
   kubectl top pods -n codigo
   ```

#### Resolution Steps

**If Database Issues:**
1. Check PostgreSQL pod status:
   ```bash
   kubectl get pods -n codigo -l app=postgres
   kubectl logs -n codigo -l app=postgres --tail=50
   ```
2. Check database connections:
   ```bash
   # Query Prometheus
   db_connections_active{service="codigo-api"}
   ```
3. Restart API pods if needed:
   ```bash
   kubectl rollout restart deployment/codigo-api -n codigo
   ```

**If NATS Issues:**
1. Check NATS pod status:
   ```bash
   kubectl get pods -n codigo -l app=nats
   kubectl logs -n codigo -l app=nats --tail=50
   ```
2. Restart NATS if needed:
   ```bash
   kubectl rollout restart deployment/nats -n codigo
   ```

**If Application Issues:**
1. Check API pod logs:
   ```bash
   kubectl logs -n codigo -l app=codigo-api --tail=100
   ```
2. Check for recent deployments:
   ```bash
   kubectl rollout history deployment/codigo-api -n codigo
   ```
3. Rollback if recent deployment:
   ```bash
   kubectl rollout undo deployment/codigo-api -n codigo
   ```

**If Resource Constraints:**
1. Scale up API deployment:
   ```bash
   kubectl scale deployment/codigo-api -n codigo --replicas=3
   ```
2. Increase resource limits in values.yaml if needed

#### Verification

1. **Monitor Error Rate:**
   ```bash
   # Should return < 0.01 (1%)
   sum(rate(http_requests_total{service=~"codigo-api", code=~"5.."}[5m])) 
   / 
   sum(rate(http_requests_total{service=~"codigo-api"}[5m]))
   ```

2. **Check Alert Status:**
   - Verify alert clears in Prometheus/Alertmanager
   - Confirm availability returns to > 99.9%

#### Escalation

- If error rate > 5% for 2 minutes → `CriticalErrorRate` alert fires
- Escalate to on-call engineer if not resolved in 15 minutes
- Escalate to team lead if SLO breach is imminent

---

### Runbook 2: High Latency Alert

**Alert:** `HighLatency`  
**Severity:** Warning  
**SLO:** Latency

#### Symptoms
- p95 latency > 500ms for 5 minutes
- Latency SLO at risk
- Slow API response times

#### Investigation Steps

1. **Check Current Latency:**
   ```bash
   # Query Prometheus for p95 latency
   histogram_quantile(0.95,
     sum(rate(http_request_duration_seconds_bucket{service=~"codigo-api"}[5m]))
     by (le, service)
   )
   ```

2. **Check Latency by Endpoint:**
   ```bash
   # Identify slow endpoints
   histogram_quantile(0.95,
     sum(rate(http_request_duration_seconds_bucket{service=~"codigo-api"}[5m]))
     by (le, route)
   )
   ```

3. **Check Database Query Performance:**
   ```bash
   # Check database latency (if PostgreSQL exporter available)
   histogram_quantile(0.95,
     sum(rate(pg_stat_statements_mean_exec_time_bucket[5m])) by (le)
   )
   ```

4. **Check Application Logs for Slow Operations:**
   ```bash
   # Query Loki for slow requests
   {namespace="codigo", app="codigo-api"} | json | duration > 0.5
   ```

5. **Check Resource Usage:**
   ```bash
   # Check CPU and memory
   kubectl top pods -n codigo
   
   # Check for CPU throttling
   kubectl describe pod <pod-name> -n codigo | grep -i throttle
   ```

6. **Check Trace Data:**
   - Open Grafana → Tempo
   - Filter by service: `codigo-api`
   - Look for slow spans (> 500ms)

#### Resolution Steps

**If Database Slow:**
1. Check active connections:
   ```bash
   # Query Prometheus
   db_connections_active{service="codigo-api"}
   ```
2. Check for long-running queries:
   ```bash
   kubectl exec -it deployment/postgres -n codigo -- \
     psql -U codigo -d codigo -c \
     "SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
      FROM pg_stat_activity 
      WHERE state = 'active' AND now() - pg_stat_activity.query_start > interval '5 seconds';"
   ```
3. Optimize slow queries or add database connection pooling

**If Application Bottleneck:**
1. Check for goroutine leaks:
   ```bash
   # Check metrics endpoint
   kubectl exec -it deployment/codigo-api -n codigo -- \
     curl http://localhost:8080/metrics | grep go_goroutines
   ```
2. Review recent code changes:
   ```bash
   kubectl rollout history deployment/codigo-api -n codigo
   ```
3. Rollback if recent deployment introduced latency:
   ```bash
   kubectl rollout undo deployment/codigo-api -n codigo
   ```

**If Resource Constraints:**
1. Check CPU usage:
   ```bash
   kubectl top pods -n codigo
   ```
2. Scale up API deployment:
   ```bash
   kubectl scale deployment/codigo-api -n codigo --replicas=3
   ```
3. Increase CPU limits in values.yaml:
   ```yaml
   resources:
     limits:
       cpu: "1000m"  # Increase from 500m
   ```

**If Network Issues:**
1. Check NATS message latency:
   ```bash
   # Check NATS metrics if available
   sum(rate(nats_core_latency_seconds[5m])) by (server)
   ```
2. Check inter-pod network latency:
   ```bash
   kubectl run -it --rm debug --image=busybox --restart=Never -n codigo -- \
     ping -c 5 codigo-api.codigo.svc.cluster.local
   ```

#### Verification

1. **Monitor Latency:**
   ```bash
   # Should return < 0.5 (500ms)
   histogram_quantile(0.95,
     sum(rate(http_request_duration_seconds_bucket{service=~"codigo-api"}[5m]))
     by (le, service)
   )
   ```

2. **Check Alert Status:**
   - Verify alert clears in Prometheus/Alertmanager
   - Confirm p95 latency returns to < 500ms

#### Escalation

- If p95 latency > 1s for 5 minutes → Escalate to on-call engineer
- If latency persists > 15 minutes → Escalate to team lead
- If SLO breach is imminent → Notify stakeholders

---

## PrometheusRule Configuration

Save the following as `k8s/apps/codigo/templates/prometheusrule-slo.yaml`:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: codigo-slo-alerts
  namespace: codigo
  labels:
    app: codigo
    prometheus: kube-prometheus-stack
    release: kube-prometheus-stack
spec:
  groups:
    - name: codigo_slo_alerts
      interval: 1m
      rules:
        - alert: HighErrorRate
          expr: |
            (
              sum(rate(http_requests_total{service=~"codigo-api", code=~"5.."}[5m])) 
              / 
              sum(rate(http_requests_total{service=~"codigo-api"}[5m]))
            ) > 0.01
          for: 5m
          labels:
            severity: warning
            slo: availability
          annotations:
            summary: "High error rate detected - Availability SLO at risk"
            description: "Error rate is {{ $value | humanizePercentage }}. SLO target: 99.9%."

        - alert: HighLatency
          expr: |
            histogram_quantile(0.95,
              sum(rate(http_request_duration_seconds_bucket{service=~"codigo-api"}[5m]))
              by (le, service)
            ) > 0.5
          for: 5m
          labels:
            severity: warning
            slo: latency
          annotations:
            summary: "High latency detected - Latency SLO at risk"
            description: "p95 latency is {{ $value | humanizeDuration }}. SLO target: ≤ 500ms."

        - alert: CriticalErrorRate
          expr: |
            (
              sum(rate(http_requests_total{service=~"codigo-api", code=~"5.."}[5m])) 
              / 
              sum(rate(http_requests_total{service=~"codigo-api"}[5m]))
            ) > 0.05
          for: 2m
          labels:
            severity: critical
            slo: availability
          annotations:
            summary: "Critical error rate - Availability SLO breached"
            description: "Error rate is {{ $value | humanizePercentage }}. SLO target: 99.9%."

        - alert: JobProcessingFailure
          expr: |
            (
              sum(rate(jobs_processed_total{service="codigo-worker", result="error"}[5m])) 
              / 
              sum(rate(jobs_processed_total{service="codigo-worker"}[5m]))
            ) > 0.1
          for: 5m
          labels:
            severity: warning
            component: worker
          annotations:
            summary: "High job processing failure rate"
            description: "Job failure rate is {{ $value | humanizePercentage }}."
```

## SLO Dashboard Queries

### Availability SLO
```promql
# Current availability (30-day window)
sum(rate(http_requests_total{service=~"codigo-api", code!~"5.."}[30d])) 
/ 
sum(rate(http_requests_total{service=~"codigo-api"}[30d]))

# Error budget remaining
1 - (
  (sum(rate(http_requests_total{service=~"codigo-api", code=~"5.."}[30d])) 
   / 
   sum(rate(http_requests_total{service=~"codigo-api"}[30d])))
  / 0.001
)
```

### Latency SLO
```promql
# Current p95 latency (30-day window)
histogram_quantile(0.95,
  sum(rate(http_request_duration_seconds_bucket{service=~"codigo-api"}[30d]))
  by (le, service)
)

# Percentage of requests meeting SLO (p95 ≤ 500ms)
sum(rate(http_request_duration_seconds_bucket{service=~"codigo-api", le="0.5"}[30d]))
/
sum(rate(http_request_duration_seconds_bucket{service=~"codigo-api"}[30d]))
```

## Summary

**SLIs:**
- ✅ Availability (99.9% target)
- ✅ Latency p95 (≤ 500ms target)
- ✅ Job Success Rate

**SLOs:**
- ✅ API Availability: 99.9%
- ✅ API Latency: p95 ≤ 500ms

**Alerts:**
- ✅ High Error Rate (Warning)
- ✅ High Latency (Warning)
- ✅ Critical Error Rate (Critical)
- ✅ Job Processing Failure (Warning)

**Runbooks:**
- ✅ High Error Rate Alert
- ✅ High Latency Alert

