# Relay

Relay is a lightweight load balancer written in Go.

It is designed to be used both as a standalone CLI application and as an embeddable Go package. The project focuses on simplicity, extensibility, and observability while providing a solid foundation for building production-grade traffic routing systems.

## Features

* HTTP load balancing
* Multiple load balancing strategies
* Backend health checks
* Prometheus metrics
* Structured logging
* YAML-based configuration
* Extensible architecture for future TCP, UDP, and gRPC support

## Project Goals

Relay is primarily a learning and engineering project aimed at exploring:

* Networking
* Reverse proxies
* Concurrency
* Distributed systems concepts
* Observability
* Infrastructure software design

The long-term goal is to provide a clean, modular, and extensible load balancing platform that developers can run as a service or integrate directly into their applications.

## Status

🚧 Relay is currently under active development.

Implemented:
- Configuration loading and validation
- Target pool management
- Strategy abstraction
- Round Robin load balancing
- Balancer engine
- Relay assembly
- HTTP reverse proxy
- Health checks
- Prometheus metrics

Planned:
- Additional balancing strategies
- And much more...

## License

MIT
