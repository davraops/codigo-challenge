package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// SLO Targets
	availabilityTarget = 0.999  // 99.9%
	latencyTargetP95  = 0.5     // 500ms in seconds
	windowDays         = 30      // 30-day window
)

type PrometheusClient struct {
	baseURL string
	client  *http.Client
}

func NewPrometheusClient(baseURL string) *PrometheusClient {
	return &PrometheusClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *PrometheusClient) Query(ctx context.Context, query string) (float64, error) {
	reqURL := fmt.Sprintf("%s/api/v1/query", p.baseURL)
	params := url.Values{}
	params.Add("query", query)

	resp, err := p.client.Get(fmt.Sprintf("%s?%s", reqURL, params.Encode()))
	if err != nil {
		return 0, fmt.Errorf("failed to query Prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("Prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Value []interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return 0, fmt.Errorf("Prometheus query failed: %s", result.Status)
	}

	if len(result.Data.Result) == 0 {
		return 0, fmt.Errorf("no data returned from query")
	}

	// Parse the value (Prometheus returns [timestamp, value])
	valueStr, ok := result.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, fmt.Errorf("invalid value format")
	}

	var value float64
	if _, err := fmt.Sscanf(valueStr, "%f", &value); err != nil {
		return 0, fmt.Errorf("failed to parse value: %w", err)
	}

	return value, nil
}

type SLOReport struct {
	SLI              string
	CurrentValue     float64
	Target           float64
	ErrorBudget      float64
	ErrorBudgetSpent float64
	ErrorBudgetLeft  float64
	BurnRate         float64
	Status           string
}

func calculateAvailabilitySLO(ctx context.Context, client *PrometheusClient) (*SLOReport, error) {
	// Calculate current availability (30-day window)
	// Availability = (non-5xx requests) / (total requests)
	query := fmt.Sprintf(`
		sum(rate(http_requests_total{service=~"codigo-api", code!~"5.."}[%dd])) 
		/ 
		sum(rate(http_requests_total{service=~"codigo-api"}[%dd]))
	`, windowDays, windowDays)

	currentAvailability, err := client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query availability: %w", err)
	}

	// Calculate error rate
	errorRate := 1 - currentAvailability

	// Error budget: 0.1% (1 - 0.999)
	errorBudget := 1 - availabilityTarget

	// Error budget spent
	errorBudgetSpent := errorRate / errorBudget

	// Error budget left
	errorBudgetLeft := 1 - errorBudgetSpent

	// Burn rate: error rate / error budget (how fast we're burning through budget)
	burnRate := errorRate / errorBudget

	status := "✅ Healthy"
	if errorBudgetSpent > 0.8 {
		status = "⚠️ Warning"
	}
	if errorBudgetSpent >= 1.0 {
		status = "❌ Breached"
	}

	return &SLOReport{
		SLI:              "Availability",
		CurrentValue:     currentAvailability,
		Target:           availabilityTarget,
		ErrorBudget:      errorBudget,
		ErrorBudgetSpent: errorBudgetSpent,
		ErrorBudgetLeft:  errorBudgetLeft,
		BurnRate:         burnRate,
		Status:           status,
	}, nil
}

func calculateLatencySLO(ctx context.Context, client *PrometheusClient) (*SLOReport, error) {
	// Calculate current p95 latency (30-day window)
	query := fmt.Sprintf(`
		histogram_quantile(0.95,
			sum(rate(http_request_duration_seconds_bucket{service=~"codigo-api"}[%dd]))
			by (le, service)
		)
	`, windowDays)

	currentLatency, err := client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query latency: %w", err)
	}

	// Calculate percentage of requests meeting SLO (p95 ≤ 500ms)
	// Query for requests below threshold (simplified: if p95 > target, estimate violations)
	meetingSLO := 1.0
	if currentLatency > latencyTargetP95 {
		// Estimate: calculate how much p95 exceeds target
		// If p95 is 600ms and target is 500ms, we estimate ~10-15% violations
		excessRatio := (currentLatency - latencyTargetP95) / latencyTargetP95
		// Cap at 5% (error budget limit)
		violationRate := excessRatio * 0.1 // Conservative estimate
		if violationRate > 0.05 {
			violationRate = 0.05
		}
		meetingSLO = 1 - violationRate
	}

	// Error budget: 5% (requests can exceed 500ms)
	errorBudget := 0.05

	// Error budget spent
	errorBudgetSpent := (1 - meetingSLO) / errorBudget

	// Error budget left
	errorBudgetLeft := 1 - errorBudgetSpent

	// Burn rate
	burnRate := (1 - meetingSLO) / errorBudget

	status := "✅ Healthy"
	if errorBudgetSpent > 0.8 {
		status = "⚠️ Warning"
	}
	if errorBudgetSpent >= 1.0 {
		status = "❌ Breached"
	}

	return &SLOReport{
		SLI:              "Latency (p95)",
		CurrentValue:     currentLatency,
		Target:           latencyTargetP95,
		ErrorBudget:      errorBudget,
		ErrorBudgetSpent: errorBudgetSpent,
		ErrorBudgetLeft:  errorBudgetLeft,
		BurnRate:         burnRate,
		Status:           status,
	}, nil
}

func printReport(reports []*SLOReport) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("SLO REPORT - Codigo Application")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Window: %d days\n", windowDays)
	fmt.Printf("Generated: %s\n\n", time.Now().Format(time.RFC3339))

	for _, report := range reports {
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("SLO: %s\n", report.SLI)
		fmt.Printf("Status: %s\n", report.Status)
		fmt.Printf("Current Value: %.4f\n", report.CurrentValue)
		fmt.Printf("Target: %.4f\n", report.Target)

		if report.SLI == "Availability" {
			fmt.Printf("Current Availability: %.2f%%\n", report.CurrentValue*100)
			fmt.Printf("Target Availability: %.2f%%\n", report.Target*100)
		} else {
			fmt.Printf("Current p95 Latency: %.0fms\n", report.CurrentValue*1000)
			fmt.Printf("Target p95 Latency: %.0fms\n", report.Target*1000)
		}

		fmt.Printf("\nError Budget:\n")
		fmt.Printf("  Total Budget: %.2f%%\n", report.ErrorBudget*100)
		fmt.Printf("  Budget Spent: %.2f%%\n", report.ErrorBudgetSpent*100)
		fmt.Printf("  Budget Left: %.2f%%\n", report.ErrorBudgetLeft*100)
		fmt.Printf("  Burn Rate: %.2fx\n", report.BurnRate)

		if report.BurnRate > 1.0 {
			daysUntilExhaustion := windowDays / report.BurnRate
			fmt.Printf("  ⚠️  At current burn rate, error budget will be exhausted in ~%.0f days\n", daysUntilExhaustion)
		}

		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 80))
}

func main() {
	var (
		prometheusURL = flag.String("prometheus-url", "http://localhost:9090", "Prometheus base URL")
		output        = flag.String("output", "text", "Output format: text or json")
	)
	flag.Parse()

	ctx := context.Background()
	client := NewPrometheusClient(*prometheusURL)

	// Calculate SLOs
	availabilityReport, err := calculateAvailabilitySLO(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating availability SLO: %v\n", err)
		os.Exit(1)
	}

	latencyReport, err := calculateLatencySLO(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating latency SLO: %v\n", err)
		os.Exit(1)
	}

	reports := []*SLOReport{availabilityReport, latencyReport}

	// Output
	if *output == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(reports); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		printReport(reports)
	}
}

