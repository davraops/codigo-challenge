# SLO Reporter

A command-line tool that queries Prometheus to calculate and report on Service Level Objectives (SLOs) for the Codigo application.

## Features

- **Availability SLO**: Calculates current availability and error budget burn rate
- **Latency SLO**: Calculates p95 latency and error budget burn rate
- **Error Budget Tracking**: Shows how much error budget has been spent
- **Burn Rate Calculation**: Estimates time until error budget exhaustion
- **Multiple Output Formats**: Text (human-readable) or JSON

## SLOs Tracked

1. **Availability SLO**
   - Target: 99.9% availability
   - Error Budget: 0.1% (43.2 minutes per month)
   - Window: 30 days

2. **Latency SLO**
   - Target: p95 ≤ 500ms
   - Error Budget: 5% of requests can exceed 500ms
   - Window: 30 days

## Installation

```bash
cd tools/slo-reporter
go mod tidy
go build -o slo-reporter
```

## Usage

### Basic Usage

```bash
# Connect to local Prometheus
./slo-reporter -prometheus-url http://localhost:9090

# Connect to Prometheus in Kubernetes (via port-forward)
kubectl port-forward svc/kube-prometheus-stack-prometheus -n observability 9090:9090
./slo-reporter -prometheus-url http://localhost:9090
```

### JSON Output

```bash
./slo-reporter -prometheus-url http://localhost:9090 -output json
```

### Example Output

```
================================================================================
SLO REPORT - Codigo Application
================================================================================
Window: 30 days
Generated: 2024-01-15T10:30:00Z

--------------------------------------------------------------------------------
SLO: Availability
Status: ✅ Healthy
Current Value: 0.9995
Target: 0.9990
Current Availability: 99.95%
Target Availability: 99.90%

Error Budget:
  Total Budget: 0.10%
  Budget Spent: 50.00%
  Budget Left: 50.00%
  Burn Rate: 0.50x

--------------------------------------------------------------------------------
SLO: Latency (p95)
Status: ✅ Healthy
Current Value: 0.4500
Target: 0.5000
Current p95 Latency: 450ms
Target p95 Latency: 500ms

Error Budget:
  Total Budget: 5.00%
  Budget Spent: 0.00%
  Budget Left: 100.00%
  Burn Rate: 0.00x

================================================================================
```

## Integration

### CI/CD Integration

Add to your deployment pipeline to verify SLOs before promoting:

```bash
# Check SLOs before deployment
if ! ./slo-reporter -prometheus-url $PROMETHEUS_URL | grep -q "✅ Healthy"; then
  echo "SLOs not met - blocking deployment"
  exit 1
fi
```

### Scheduled Reports

Run as a cron job to generate daily SLO reports:

```bash
# Daily SLO report
0 9 * * * /path/to/slo-reporter -prometheus-url http://prometheus:9090 >> /var/log/slo-reports.log
```

### Alerting Integration

Use with alerting systems to trigger when SLOs are at risk:

```bash
# Check if any SLO is breached
./slo-reporter -prometheus-url $PROMETHEUS_URL -output json | \
  jq '.[] | select(.errorBudgetSpent >= 1.0)'
```

## Prometheus Queries Used

### Availability
```promql
sum(rate(http_requests_total{service=~"codigo-api", code!~"5.."}[30d])) 
/ 
sum(rate(http_requests_total{service=~"codigo-api"}[30d]))
```

### Latency (p95)
```promql
histogram_quantile(0.95,
  sum(rate(http_request_duration_seconds_bucket{service=~"codigo-api"}[30d]))
  by (le, service)
)
```

## Error Budget Calculation

**Error Budget = 1 - SLO Target**

**Error Budget Spent = (Current Error Rate) / (Error Budget)**

**Burn Rate = (Current Error Rate) / (Error Budget)**

**Time Until Exhaustion = (Window Days) / (Burn Rate)**

## Troubleshooting

### "No data returned from query"

- Verify Prometheus is accessible
- Check that metrics are being scraped (visit Prometheus UI)
- Verify service labels match: `service=~"codigo-api"`

### "Prometheus returned status 400"

- Check Prometheus query syntax
- Verify Prometheus version compatibility
- Check Prometheus logs for errors

### Connection Timeout

- Verify network connectivity to Prometheus
- Check firewall rules
- Use port-forward if accessing from outside cluster

## Future Enhancements

- [ ] Support for custom SLO definitions via config file
- [ ] Historical trend analysis
- [ ] Multi-service SLO tracking
- [ ] Integration with alerting systems (PagerDuty, Slack)
- [ ] Webhook notifications for SLO breaches
- [ ] Export to monitoring dashboards

