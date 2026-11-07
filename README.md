# Relay

Relay is a lightweight load balancer written in Go.

It runs as a standalone HTTP service. The packages live under `internal/`, so Relay is not currently a public embeddable Go API. The project focuses on simplicity, extensibility, and observability.

## Features

* HTTP load balancing
* Round robin, weighted round robin, least connections, least response time, and consistent hashing
* Active TCP/HTTP health checks with passive failure detection
* Retries for GET/HEAD requests on connection failures
* Per-IP rate limiting and max-in-flight limits
* Graceful shutdown and SIGHUP configuration reload
* Prometheus metrics
* Structured logging
* Admin, liveness, and readiness endpoints
* YAML-based configuration
* Docker Compose example with Prometheus

## Project Goals

Relay is primarily a learning and engineering project aimed at exploring:

* Networking
* Reverse proxies
* Concurrency
* Distributed systems concepts
* Observability
* Infrastructure software design

The long-term goal is to provide a clean, modular, and extensible load balancing platform.

## Status

🚧 Relay is currently under active development.

Implemented:
- Configuration loading and validation
- Target pool management and health tracking
- All five balancing strategies
- Balancer engine and HTTP reverse proxy
- Health checks, retries, rate limiting, and reloads
- Prometheus metrics and structured logging
- `/admin/targets`, `/healthz`, and `/readyz`

Planned:
- Additional transports such as TCP, UDP, and gRPC
- A stable public Go API

## License

MIT
