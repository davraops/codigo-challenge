module codigo/worker

go 1.22

require (
  github.com/jackc/pgx/v5 v5.7.1
  github.com/nats-io/nats.go v1.36.0
  github.com/prometheus/client_golang v1.20.4
  go.opentelemetry.io/otel v1.31.0
  go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.31.0
  go.opentelemetry.io/otel/sdk v1.31.0
)
