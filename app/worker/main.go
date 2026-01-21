package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	jobsProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "jobs_processed_total",
		Help: "Total jobs processed",
	}, []string{"service", "result"})

	jobLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "job_processing_duration_seconds",
		Help:    "Job processing duration",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"service"})

	dbConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "db_connections_active",
		Help: "Active database connections",
	}, []string{"service"})

	natsMessagesReceived = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_messages_received_total",
		Help: "Total NATS messages received",
	}, []string{"service", "subject"})
)

func main() {
	serviceName := getenv("SERVICE_NAME", "codigo-worker")

	// Initialize structured logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// Register Prometheus metrics
	prometheus.MustRegister(jobsProcessed, jobLatency, dbConnections, natsMessagesReceived)

	ctx := context.Background()

	// Initialize OpenTelemetry
	shutdown := initOTel(ctx, serviceName)
	defer shutdown()

	// Initialize database
	db := mustDB(ctx)
	defer db.Close()

	// Initialize NATS
	nc := mustNATS()
	defer nc.Close()

	// Start metrics HTTP server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("ok"))
		}))
		logger.Info("metrics server starting", zap.String("address", ":8080"))
		if err := http.ListenAndServe(":8080", nil); err != nil {
			logger.Fatal("metrics server failed", zap.Error(err))
		}
	}()

	// Start background goroutine to update DB connection metrics
	go updateDBMetrics(db, serviceName)

	// Subscribe to jobs
	_, err = nc.Subscribe("jobs", func(m *nats.Msg) {
		processJob(m, db, serviceName, logger)
	})
	if err != nil {
		logger.Fatal("failed to subscribe to jobs", zap.Error(err))
	}

	logger.Info("worker running", zap.String("subject", "jobs"))
	select {}
}

func processJob(m *nats.Msg, db *pgxpool.Pool, serviceName string, logger *zap.Logger) {
	start := time.Now()
	jobID := string(m.Data)

	// Extract trace context from NATS headers
	propagator := otel.GetTextMapPropagator()
	ctx := propagator.Extract(context.Background(), natsHeaderCarrier(m.Header))

	// Start span with extracted context
	tr := otel.Tracer("codigo-worker")
	ctx, span := tr.Start(ctx, "processJob")
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	span.SetAttributes(
		attribute.String("job.id", jobID),
		attribute.String("nats.subject", m.Subject),
	)

	logger.Info("processing job",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("job_id", jobID))

	natsMessagesReceived.WithLabelValues(serviceName, m.Subject).Inc()

	// Simulate work
	time.Sleep(150 * time.Millisecond)

	// Update job status
	_, err := db.Exec(ctx, `UPDATE jobs SET status='done' WHERE id=$1`, jobID)
	if err != nil {
		logger.Error("database error - update job",
			zap.String("trace_id", traceID),
			zap.String("job_id", jobID),
			zap.Error(err))
		span.RecordError(err)
		jobsProcessed.WithLabelValues(serviceName, "error").Inc()
		return
	}

	duration := time.Since(start)
	jobsProcessed.WithLabelValues(serviceName, "ok").Inc()
	jobLatency.WithLabelValues(serviceName).Observe(duration.Seconds())

	span.SetAttributes(
		attribute.String("job.status", "done"),
		attribute.Float64("job.duration_ms", float64(duration.Milliseconds())),
	)

	logger.Info("job processed successfully",
		zap.String("trace_id", traceID),
		zap.String("job_id", jobID),
		zap.Duration("duration", duration))
}

func updateDBMetrics(db *pgxpool.Pool, serviceName string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := db.Stat()
		dbConnections.WithLabelValues(serviceName).Set(float64(stats.AcquiredConns()))
	}
}

// natsHeaderCarrier adapts NATS headers to OpenTelemetry propagation
type natsHeaderCarrier nats.Header

func (c natsHeaderCarrier) Get(key string) string {
	vals := nats.Header(c).Values(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (c natsHeaderCarrier) Set(key, value string) {
	nats.Header(c).Set(key, value)
}

func (c natsHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

func mustDB(ctx context.Context) *pgxpool.Pool {
	host := getenv("POSTGRES_HOST", "localhost")
	port := getenv("POSTGRES_PORT", "5432")
	db := getenv("POSTGRES_DB", "codigo")
	user := getenv("POSTGRES_USER", "codigo")
	// POSTGRES_PASSWORD must be set via environment variable (Kubernetes Secret)
	// No default value for security - fail if not set
	pass := os.Getenv("POSTGRES_PASSWORD")
	if pass == "" {
		panic("POSTGRES_PASSWORD environment variable is required")
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, db)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	return pool
}

func mustNATS() *nats.Conn {
	url := getenv("NATS_URL", "nats://127.0.0.1:4222")
	nc, err := nats.Connect(url, nats.Timeout(2*time.Second))
	if err != nil {
		panic(err)
	}
	return nc
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
