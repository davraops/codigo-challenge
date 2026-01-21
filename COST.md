# Cost Awareness and Optimization

This document describes the cost drivers, optimization choices, and recommendations for cost monitoring in production.

## Biggest Cost Drivers

### 1. GKE Cluster Nodes (Primary Cost Driver)

**Current Configuration:**
- Machine Type: `e2-standard-2` (2 vCPU, 8GB RAM)
- Node Count: Configurable per environment (default: 1-5 nodes)
- Estimated Cost: ~$50-250/month per node (depending on region and usage)

**Cost Breakdown:**
- **Dev Environment:** 1 node × $50/month = **~$50/month**
- **QA Environment:** 1-2 nodes × $50/month = **~$50-100/month**
- **Preprod Environment:** 2-3 nodes × $50/month = **~$100-150/month**
- **Prod Environment:** 3-5 nodes × $50/month = **~$150-250/month**

**Total GKE Cost (4 environments):** ~$350-550/month

### 2. Persistent Storage

**Current Configuration:**
- Prometheus: 50Gi per environment
- Grafana: 10Gi per environment
- Loki: 20Gi per environment
- **Total per environment:** ~80Gi
- **Total (4 environments):** ~320Gi

**Cost Breakdown:**
- Standard persistent disk: ~$0.17/GB/month
- **Total Storage Cost:** ~$54/month for all environments

### 3. Observability Stack Resources

**Current Configuration:**

**Prometheus (per environment):**
- Memory: 2Gi request, 4Gi limit
- CPU: 1 CPU request, 2 CPU limit
- Estimated: ~$15-20/month per environment

**Grafana (per environment):**
- Memory: ~512Mi
- CPU: ~500m
- Estimated: ~$5-10/month per environment

**Loki (per environment):**
- Memory: 512Mi request, 1Gi limit
- CPU: 500m request, 1 CPU limit
- Estimated: ~$5-10/month per environment

**Tempo (per environment):**
- Memory: 512Mi request, 1Gi limit
- CPU: 500m request, 1 CPU limit
- Estimated: ~$5-10/month per environment

**Total Observability Stack (per environment):** ~$30-50/month  
**Total (4 environments):** ~$120-200/month

### 4. Application Resources

**Current Configuration:**

**API (2 replicas per environment):**
- Memory: 128Mi request, 512Mi limit per pod
- CPU: 100m request, 500m limit per pod
- Estimated: ~$2-5/month per environment

**Worker (1 replica per environment):**
- Memory: 128Mi request, 512Mi limit
- CPU: 100m request, 500m limit
- Estimated: ~$1-3/month per environment

**PostgreSQL (1 per environment):**
- Memory: ~256Mi-512Mi
- CPU: ~500m-1 CPU
- Estimated: ~$5-10/month per environment

**NATS (1 per environment):**
- Memory: ~128Mi-256Mi
- CPU: ~200m-500m
- Estimated: ~$2-5/month per environment

**Total Application Resources (per environment):** ~$10-23/month  
**Total (4 environments):** ~$40-92/month

### 5. Network Egress

**Cost Drivers:**
- Inter-region traffic
- External API calls
- Image pulls from Artifact Registry
- Estimated: ~$10-50/month (highly variable)

### 6. Artifact Registry

**Cost Drivers:**
- Storage: ~$0.10/GB/month
- Operations: Minimal cost
- Estimated: ~$1-5/month

## Total Estimated Monthly Cost

| Component | Dev | QA | Preprod | Prod | Total |
|-----------|-----|----|---------|------|-------|
| GKE Nodes | $50 | $50 | $100 | $150 | $350 |
| Storage | $14 | $14 | $14 | $14 | $56 |
| Observability | $30 | $30 | $30 | $50 | $140 |
| Application | $10 | $10 | $15 | $25 | $60 |
| Network | $5 | $5 | $10 | $20 | $40 |
| **Total** | **$109** | **$109** | **$169** | **$259** | **~$646/month** |

**Note:** Costs are estimates and vary by region, usage patterns, and GCP pricing.

## Cost Optimization Choices

### 1. GKE Node Optimization

**Choices Made:**
- ✅ **Machine Type:** `e2-standard-2` (balanced cost/performance)
- ✅ **Node Count:** Configurable per environment (1-5 nodes)
- ✅ **Validation:** Maximum 5 nodes per environment to prevent runaway costs

**Optimization:**
```hcl
# infra/terraform/variables.tf
variable "gke_num_nodes" {
  description = "Number of nodes in the GKE cluster"
  type        = number
  default     = 1
  validation {
    condition     = var.gke_num_nodes > 0 && var.gke_num_nodes <= 5
    error_message = "Number of nodes must be between 1 and 5."
  }
}
```

### 2. Storage Optimization

**Choices Made:**
- ✅ **Retention Periods:**
  - Prometheus: 30 days (reduced from default 15 days)
  - Loki: 7 days (reasonable for log analysis)
  - Tempo: 48 hours (sufficient for trace debugging)
- ✅ **Storage Classes:** Standard (cost-effective, can upgrade to SSD if needed)
- ✅ **Storage Sizing:** Right-sized based on retention and ingestion rates

**Storage per Environment:**
- Prometheus: 50Gi (30-day retention)
- Grafana: 10Gi (dashboard and user data)
- Loki: 20Gi (7-day retention)

### 3. Resource Requests and Limits

**Choices Made:**
- ✅ **Right-sized Resources:** Requests set to actual needs, limits prevent resource exhaustion
- ✅ **CPU/Memory Ratios:** Balanced ratios to avoid waste
- ✅ **Replica Counts:** Minimal replicas (API: 2, Worker: 1) for HA without over-provisioning

**Example:**
```yaml
resources:
  requests:
    cpu: "100m"      # Minimal request
    memory: "128Mi"  # Minimal request
  limits:
    cpu: "500m"      # Prevents runaway usage
    memory: "512Mi"  # Prevents memory leaks
```

### 4. Observability Stack Optimization

**Choices Made:**
- ✅ **Loki Single Binary Mode:** Lower resource usage than distributed mode
- ✅ **Prometheus Resource Limits:** Prevent memory growth issues
- ✅ **Tempo Local Storage:** Avoids additional object storage costs
- ✅ **OpenTelemetry Collector:** Batch processing reduces overhead

### 5. Multi-Environment Strategy

**Choices Made:**
- ✅ **Separate Projects:** Isolates costs per environment
- ✅ **Environment-Specific Scaling:** Dev/QA use minimal resources, Prod scales as needed
- ✅ **Shared State Bucket:** Single Terraform state project reduces overhead

### 6. Image Optimization

**Choices Made:**
- ✅ **Distroless Images:** Smaller images = faster pulls = lower network costs
- ✅ **Multi-stage Builds:** Reduces final image size
- ✅ **Artifact Registry:** Regional storage reduces egress costs

## What to Add for Cost Monitoring in Production

### 1. GCP Cost Monitoring

**Recommended Tools:**

**a) GCP Billing Budgets and Alerts**
```yaml
# Create budgets per project/environment
- Budget for Dev project: $150/month
- Budget for QA project: $150/month
- Budget for Preprod project: $200/month
- Budget for Prod project: $300/month
- Budget for State project: $50/month
```

**b) Cost Allocation Labels**
- Add labels to all resources:
  - `environment`: dev, qa, preprod, prod
  - `team`: platform, engineering
  - `cost-center`: infrastructure, application
  - `project`: codigo

**c) Cost Export to BigQuery**
- Enable billing export to BigQuery
- Create dashboards in Data Studio/Looker
- Set up automated cost reports

### 2. Kubernetes Cost Monitoring

**Recommended Tools:**

**a) Kubecost**
```yaml
# Deploy Kubecost for Kubernetes cost visibility
apiVersion: v1
kind: Namespace
metadata:
  name: kubecost
---
# Kubecost provides:
- Cost per namespace
- Cost per pod/deployment
- Resource efficiency metrics
- Cost allocation by labels
- Recommendations for rightsizing
```

**b) Prometheus Cost Metrics**
- Export GCP billing data to Prometheus
- Create cost dashboards in Grafana
- Alert on cost anomalies

**c) Resource Usage Monitoring**
- Track actual vs requested resources
- Identify over-provisioned pods
- Right-size based on actual usage

### 3. Cost Alerts

**Recommended Alerts:**

**a) Budget Threshold Alerts**
```yaml
# Alert when 50% of budget consumed
- Alert: Budget50Percent
  Condition: cost > budget * 0.5
  Severity: warning

# Alert when 80% of budget consumed
- Alert: Budget80Percent
  Condition: cost > budget * 0.8
  Severity: warning

# Alert when 100% of budget consumed
- Alert: BudgetExceeded
  Condition: cost > budget
  Severity: critical
```

**b) Anomaly Detection**
```yaml
# Alert on unexpected cost spikes
- Alert: CostSpike
  Condition: daily_cost > avg_daily_cost * 1.5
  Severity: warning
```

**c) Resource Waste Alerts**
```yaml
# Alert on low resource utilization
- Alert: LowCPUUtilization
  Condition: avg_cpu_usage < requested_cpu * 0.3
  Severity: info
  Action: Recommend rightsizing

# Alert on low memory utilization
- Alert: LowMemoryUtilization
  Condition: avg_memory_usage < requested_memory * 0.3
  Severity: info
  Action: Recommend rightsizing
```

### 4. Cost Optimization Automation

**Recommended Implementations:**

**a) Cluster Autoscaler**
```yaml
# Enable GKE cluster autoscaler
resource "google_container_node_pool" "primary_nodes" {
  autoscaling {
    min_node_count = 1
    max_node_count = 5
  }
}
```
- Automatically scales nodes based on demand
- Reduces costs during low-traffic periods

**b) Horizontal Pod Autoscaler (HPA)**
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: codigo-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: codigo-api
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```
- Scales pods based on CPU/memory usage
- Reduces costs by scaling down during low traffic

**c) Vertical Pod Autoscaler (VPA)**
```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: codigo-api-vpa
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: codigo-api
  updatePolicy:
    updateMode: "Off"  # Recommendations only
```
- Provides recommendations for right-sizing
- Can auto-adjust requests/limits (use with caution)

**d) Pod Disruption Budgets**
```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: codigo-api-pdb
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: codigo-api
```
- Prevents unnecessary pod restarts
- Reduces resource churn and associated costs

### 5. Cost Reporting and Dashboards

**Recommended Dashboards:**

**a) GCP Cost Dashboard (Grafana)**
- Total cost by project
- Cost by service (GKE, Storage, Network)
- Cost trends over time
- Cost by environment
- Forecasted costs

**b) Kubernetes Cost Dashboard (Kubecost)**
- Cost per namespace
- Cost per deployment/pod
- Resource efficiency
- Waste identification
- Recommendations

**c) Resource Utilization Dashboard**
- CPU/Memory usage vs requests
- Storage usage trends
- Network egress trends
- Pod count trends

### 6. Cost Governance

**Recommended Practices:**

**a) Cost Reviews**
- Weekly cost reviews for non-prod environments
- Monthly cost reviews for production
- Quarterly cost optimization reviews

**b) Resource Tagging Policy**
- Mandatory labels for all resources
- Cost allocation by team/project
- Chargeback/showback reports

**c) Approval Workflows**
- Require approval for resource increases
- Budget approval for new environments
- Cost impact assessment for new features

**d) Cost Optimization Sprints**
- Regular optimization sprints
- Focus on high-cost areas
- Implement recommendations from monitoring tools

## Cost Optimization Roadmap

### Phase 1: Immediate (Week 1-2)
- [ ] Set up GCP billing budgets and alerts
- [ ] Deploy Kubecost
- [ ] Add cost allocation labels to all resources
- [ ] Create cost dashboards in Grafana

### Phase 2: Short-term (Month 1)
- [ ] Enable cluster autoscaler
- [ ] Implement HPA for API and Worker
- [ ] Right-size resources based on actual usage
- [ ] Set up cost anomaly detection

### Phase 3: Medium-term (Month 2-3)
- [ ] Implement VPA recommendations
- [ ] Optimize storage retention policies
- [ ] Review and optimize observability stack resources
- [ ] Implement cost governance processes

### Phase 4: Long-term (Ongoing)
- [ ] Regular cost optimization reviews
- [ ] Spot/Preemptible instances for non-prod (if applicable)
- [ ] Reserved instances for predictable workloads
- [ ] Multi-region cost optimization

## Cost Monitoring Tools Comparison

| Tool | Purpose | Cost | Setup Complexity |
|------|---------|------|------------------|
| **GCP Billing** | Native GCP cost tracking | Free | Low |
| **Kubecost** | Kubernetes cost allocation | Free tier available | Medium |
| **Prometheus + Custom Metrics** | Cost metrics in Prometheus | Free | High |
| **BigQuery + Data Studio** | Cost analytics and reporting | Pay per query | Medium |
| **CloudHealth** | Multi-cloud cost management | Paid | Low |
| **Cloudability** | Cost optimization platform | Paid | Low |

## Recommendations Summary

### Immediate Actions
1. ✅ **Set up GCP billing budgets** - Prevent cost overruns
2. ✅ **Deploy Kubecost** - Get Kubernetes cost visibility
3. ✅ **Add cost labels** - Enable cost allocation
4. ✅ **Create cost dashboards** - Monitor costs in real-time

### Short-term Optimizations
1. ✅ **Enable autoscaling** - Reduce costs during low traffic
2. ✅ **Right-size resources** - Based on actual usage data
3. ✅ **Optimize storage** - Adjust retention based on needs
4. ✅ **Review observability stack** - Optimize resource allocation

### Long-term Strategy
1. ✅ **Cost governance** - Regular reviews and optimization
2. ✅ **Reserved instances** - For predictable workloads
3. ✅ **Multi-region optimization** - Reduce network costs
4. ✅ **Continuous optimization** - Ongoing cost reduction

## Estimated Cost Savings Potential

With optimizations implemented:

| Optimization | Potential Savings |
|-------------|------------------|
| Cluster Autoscaler | 20-30% (non-prod environments) |
| HPA for API/Worker | 10-20% (scale down during low traffic) |
| Right-sizing Resources | 15-25% (reduce over-provisioning) |
| Storage Optimization | 10-15% (adjust retention) |
| **Total Potential Savings** | **~40-60%** |

**Optimized Monthly Cost:** ~$260-390/month (down from ~$646/month)

## Conclusion

The current implementation includes several cost optimization choices:
- ✅ Configurable node counts with maximum limits
- ✅ Right-sized resource requests and limits
- ✅ Optimized storage retention periods
- ✅ Efficient observability stack configuration
- ✅ Multi-environment cost isolation

**Next Steps:**
1. Implement cost monitoring (Kubecost, GCP budgets)
2. Enable autoscaling (Cluster Autoscaler, HPA)
3. Right-size based on actual usage
4. Establish cost governance processes

With proper cost monitoring and optimization, the infrastructure can achieve **40-60% cost reduction** while maintaining performance and reliability.
