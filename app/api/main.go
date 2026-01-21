package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
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
	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"service", "route", "method", "code"})

	httpLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"service", "route", "method"})

	dbConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "db_connections_active",
		Help: "Active database connections",
	}, []string{"service"})

	natsMessagesPublished = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_messages_published_total",
		Help: "Total NATS messages published",
	}, []string{"service", "subject"})
)

type Server struct {
	db     *pgxpool.Pool
	nats   *nats.Conn
	logger *zap.Logger
}

func main() {
	serviceName := getenv("SERVICE_NAME", "codigo-api")

	// Initialize structured logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// Register Prometheus metrics
	prometheus.MustRegister(httpRequests, httpLatency, dbConnections, natsMessagesPublished)

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

	s := &Server{db: db, nats: nc, logger: logger}

	// Start background goroutine to update DB connection metrics
	go s.updateDBMetrics(serviceName)

	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	r.Get("/readyz", s.readyz)
	r.Get("/v1/jobs", s.createJob)
	r.Handle("/metrics", promhttp.Handler())

	addr := ":8080"
	logger.Info("api server starting", zap.String("address", addr))
	if err := http.ListenAndServe(addr, instrument(serviceName, logger, r)); err != nil {
		logger.Fatal("api server failed", zap.Error(err))
	}
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer cancel()

	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()

	if err := s.db.Ping(ctx); err != nil {
		s.logger.Warn("readiness check failed - database",
			zap.String("trace_id", traceID),
			zap.Error(err))
		http.Error(w, "db not ready", 503)
		return
	}
	if !s.nats.IsConnected() {
		s.logger.Warn("readiness check failed - nats",
			zap.String("trace_id", traceID))
		http.Error(w, "nats not ready", 503)
		return
	}
	w.WriteHeader(200)
	w.Write([]byte("ready"))
}

func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tr := otel.Tracer("codigo-api")
	ctx, span := tr.Start(ctx, "createJob")
	defer span.End()

	// Get trace ID for logging
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	id := fmt.Sprintf("job_%d", time.Now().UnixNano())
	span.SetAttributes(
		attribute.String("job.id", id),
		attribute.String("http.method", r.Method),
		attribute.String("http.route", r.URL.Path),
	)

	s.logger.Info("creating job",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("job_id", id))

	// Create table if not exists
	_, err := s.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS jobs (id text primary key, created_at timestamptz default now(), status text default 'queued');`)
	if err != nil {
		s.logger.Error("database error - create table",
			zap.String("trace_id", traceID),
			zap.String("job_id", id),
			zap.Error(err))
		span.RecordError(err)
		http.Error(w, "db error", 500)
		return
	}

	// Insert job
	_, err = s.db.Exec(ctx, `INSERT INTO jobs (id) VALUES ($1) ON CONFLICT DO NOTHING`, id)
	if err != nil {
		s.logger.Error("database error - insert job",
			zap.String("trace_id", traceID),
			zap.String("job_id", id),
			zap.Error(err))
		span.RecordError(err)
		http.Error(w, "db insert error", 500)
		return
	}

	// Publish to NATS with trace context propagation
	headers := make(nats.Header)
	headers.Set("traceparent", fmt.Sprintf("00-%s-%s-01", traceID, spanID))
	
	if err := s.nats.PublishMsg(&nats.Msg{
		Subject: "jobs",
		Data:    []byte(id),
		Header:  headers,
	}); err != nil {
		s.logger.Error("nats publish error",
			zap.String("trace_id", traceID),
			zap.String("job_id", id),
			zap.Error(err))
		span.RecordError(err)
		http.Error(w, "nats publish error", 500)
		return
	}

	natsMessagesPublished.WithLabelValues("codigo-api", "jobs").Inc()

	s.logger.Info("job created successfully",
		zap.String("trace_id", traceID),
		zap.String("job_id", id))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"job_id": id})
}

func mustDB(ctx context.Context) *pgxpool.Pool {
	host := getenv("POSTGRES_HOST", "localhost")
	port := getenv("POSTGRES_PORT", "5432")
	db := getenv("POSTGRES_DB", "codigo")
	user := getenv("POSTGRES_USER", "codigo")
	pass := getenv("POSTGRES_PASSWORD", "codigo")

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

func instrument(service string, logger *zap.Logger, next http.Handler) http.Handler {
	propagator := otel.GetTextMapPropagator()
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from HTTP headers
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		
		// Start span
		tr := otel.Tracer("codigo-api")
		ctx, span := tr.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		defer span.End()

		// Add trace context to request
		r = r.WithContext(ctx)

		route := r.URL.Path
		method := r.Method
		traceID := span.SpanContext().TraceID().String()

		start := time.Now()
		rr := &respRecorder{ResponseWriter: w, code: 200}

		next.ServeHTTP(rr, r)

		duration := time.Since(start)
		code := fmt.Sprintf("%d", rr.code)
		
		// Update metrics
		httpRequests.WithLabelValues(service, route, method, code).Inc()
		httpLatency.WithLabelValues(service, route, method).Observe(duration.Seconds())
		
		// Add span attributes
		span.SetAttributes(
			attribute.String("http.method", method),
			attribute.String("http.route", route),
			attribute.Int("http.status_code", rr.code),
			attribute.Float64("http.duration_ms", float64(duration.Milliseconds())),
		)

		// Structured logging
		logger.Info("http request",
			zap.String("trace_id", traceID),
			zap.String("method", method),
			zap.String("route", route),
			zap.Int("status_code", rr.code),
			zap.Duration("duration", duration),
		)
	})
}

func (s *Server) updateDBMetrics(serviceName string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := s.db.Stat()
		dbConnections.WithLabelValues(serviceName).Set(float64(stats.AcquiredConns()))
	}
}

type respRecorder struct {
	http.ResponseWriter
	code int
}

func (r *respRecorder) WriteHeader(statusCode int) {
	r.code = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
