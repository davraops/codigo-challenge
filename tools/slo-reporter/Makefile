.PHONY: build test clean run

build:
	go build -o slo-reporter main.go

test:
	go test -v ./...

clean:
	rm -f slo-reporter

run: build
	./slo-reporter -prometheus-url http://localhost:9090

install:
	go install .

help:
	@echo "Available targets:"
	@echo "  build   - Build the slo-reporter binary"
	@echo "  test    - Run tests"
	@echo "  clean   - Remove built binary"
	@echo "  run     - Build and run with default Prometheus URL"
	@echo "  install - Install to GOPATH/bin"

