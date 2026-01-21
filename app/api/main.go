package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
)

var (
	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"service", "route", "method", "code"})

	httpLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "route", "method"})
)

type Server struct {
	db   *pgxpool.Pool
	nats *nats.Conn
}

func main() {
	serviceName := getenv("SERVICE_NAME", "codigo-api")

	prometheus.MustRegister(httpRequests, httpLatency)

	ctx := context.Background()

	shutdown := initOTel(ctx, serviceName)
	defer shutdown()

	db := mustDB(ctx)
	defer db.Close()

	nc := mustNATS()
	defer nc.Close()

	s := &Server{db: db, nats: nc}

	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	r.Get("/readyz", s.readyz)
	r.Get("/v1/jobs", s.createJob)
	r.Handle("/metrics", promhttp.Handler())

	addr := ":8080"
	log.Printf("api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, instrument(serviceName, r)))
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		http.Error(w, "db not ready", 503)
		return
	}
	if !s.nats.IsConnected() {
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

	id := fmt.Sprintf("job_%d", time.Now().UnixNano())
	span.SetAttributes(attribute.String("job.id", id))

	_, err := s.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS jobs (id text primary key, created_at timestamptz default now(), status text default 'queued');`)
	if err != nil {
		http.Error(w, "db error", 500)
		return
	}

	_, err = s.db.Exec(ctx, `INSERT INTO jobs (id) VALUES ($1) ON CONFLICT DO NOTHING`, id)
	if err != nil {
		http.Error(w, "db insert error", 500)
		return
	}

	if err := s.nats.Publish("jobs", []byte(id)); err != nil {
		http.Error(w, "nats publish error", 500)
		return
	}

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

func instrument(service string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := r.URL.Path
		method := r.Method

		start := time.Now()
		rr := &respRecorder{ResponseWriter: w, code: 200}

		next.ServeHTTP(rr, r)

		code := fmt.Sprintf("%d", rr.code)
		httpRequests.WithLabelValues(service, route, method, code).Inc()
		httpLatency.WithLabelValues(service, route, method).Observe(time.Since(start).Seconds())
	})
}

type respRecorder struct {
	http.ResponseWriter
	code int
}

func (r *respRecorder) WriteHeader(statusCode int) {
	r.code = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
