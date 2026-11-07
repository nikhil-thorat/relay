.SILENT:

.PHONY: bench bench-smoke bench-micro

BINARY=relay
CMD=./cmd/relay

build:
	go build -o bin/$(BINARY) $(CMD)

run:
	go run $(CMD)

test:
	go test -v ./...

test-race:
	go test -race ./...

bench:
	./bench/run.sh all

bench-smoke:
	./bench/run.sh smoke

bench-micro:
	go test -run '^$$' -bench . -benchmem -count=5 ./internal/balancer ./internal/proxy

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet

clean:
	rm -rf bin

all: clean lint test build
