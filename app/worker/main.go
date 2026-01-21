package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var (
	jobsProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "jobs_processed_total",
		Help: "Total jobs processed",
	}, []string{"service", "result"})

	jobLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "job_processing_duration_seconds",
		Help:    "Job processing duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"service"})
)

func main() {
	serviceName := getenv("SERVICE_NAME", "codigo-worker")
	prometheus.MustRegister(jobsProcessed, jobLatency)

	ctx := context.Background()
	shutdown := initOTel(ctx, serviceName)
	defer shutdown()

	db := mustDB(ctx)
	defer db.Close()

	nc := mustNATS()
	defer nc.Close()

	_, err := nc.Subscribe("jobs", func(m *nats.Msg) {
		start := time.Now()
		jobID := string(m.Data)

		tr := otel.Tracer("codigo-worker")
		ctx, span := tr.Start(context.Background(), "processJob")
		span.SetAttributes(attribute.String("job.id", jobID))
		defer span.End()

		// simulate work
		time.Sleep(150 * time.Millisecond)

		_, err := db.Exec(ctx, `UPDATE jobs SET status='done' WHERE id=$1`, jobID)
		if err != nil {
			log.Printf("db update error: %v", err)
			jobsProcessed.WithLabelValues(serviceName, "error").Inc()
			return
		}

		jobsProcessed.WithLabelValues(serviceName, "ok").Inc()
		jobLatency.WithLabelValues(serviceName).Observe(time.Since(start).Seconds())
	})
	if err != nil {
		panic(err)
	}

	log.Printf("worker running, subscribed to jobs")
	select {}
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
