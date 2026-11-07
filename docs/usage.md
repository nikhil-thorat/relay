# Usage and Configuration

This guide explains how to run Relay as a standalone CLI service and how to configure it.

Relay is currently a standalone HTTP service. The Go packages live under `internal/`, so there is no stable public embedding API yet.

## Requirements

- Go 1.26.3 or newer for local builds
- Docker and Docker Compose for the example stack
- At least one HTTP backend target

Relay forwards requests over plain HTTP to each target. It does not terminate TLS or connect to HTTPS upstreams yet.

## Build and Run

### Build a binary

```bash
make build
```

The binary is written to `bin/relay`.

You can also build it directly:

```bash
go build -o bin/relay ./cmd/relay
```

### Run with the repository config

```bash
./bin/relay -config relay.yml
```

The repository config starts Relay on `:8080`, uses two local targets on ports `9001` and `9002`, enables TCP health checks, and exposes metrics on `:9090`.

Start the example backends in separate terminals:

```bash
go run ./examples/backends -port 9001 -id api_1
go run ./examples/backends -port 9002 -id api_2
```

Then send a request:

```bash
curl http://localhost:8080/
```

### Run from source

```bash
go run ./cmd/relay -config relay.yml
```

### Run with the Docker Compose example

```bash
docker compose -f examples/docker-compose.yml up --build
```

The stack provides:

| Service | Address |
| --- | --- |
| Relay | `http://localhost:8080` |
| Relay metrics | `http://localhost:9090/metrics` |
| Prometheus | `http://localhost:9091` |
| Backends | `api_1:9001`, `api_2:9002`, `api_3:9003` inside the compose network |

To switch strategies, point the relay service at another config:

```bash
RELAY_CONFIG=/configs/wrr.yml docker compose -f examples/docker-compose.yml up -d relay
```

Available example configs:

- `examples/strategies/rr.yml`
- `examples/strategies/wrr.yml`
- `examples/strategies/least-conn.yml`
- `examples/strategies/lrt.yml`
- `examples/strategies/consistent-hash.yml`

## CLI Reference

```text
relay -config <path>
```

| Flag | Default | Description |
| --- | --- | --- |
| `-config` | `relay.yml` | Path to the YAML configuration file |

Relay handles these signals:

| Signal | Behavior |
| --- | --- |
| `SIGHUP` | Reload the active config file |
| `SIGINT` | Graceful shutdown |
| `SIGTERM` | Graceful shutdown |

The process exits with status `1` when startup or shutdown returns an error. A reload error is logged and the old configuration stays active.

## Configuration File

Relay reads a single YAML file. The loader applies defaults, then validates the result.

Important validation rules:

- Unknown YAML fields are rejected.
- At least one target is required.
- Target IDs must be unique.
- Target addresses must use `host:port` and a port in the range `1..65535`.
- Server addresses must not be empty.
- Metrics must have an address when enabled.
- Health checks need a positive interval and timeout when enabled.
- Health mode must be `tcp` or `http`.
- Strategy type must be one of the supported names.
- Log level and format must be supported.
- Numeric limits and retry attempts must not be negative.

Durations use Go duration syntax: `250ms`, `5s`, `1m`, `1h30m`.

### Full example

```yaml
server:
  address: ":42069"
  read_header_timeout: "5s"
  read_timeout: "10s"
  write_timeout: "10s"
  idle_timeout: "60s"
  shutdown_timeout: "5s"
  max_inflight: 0
  rate_limit_rps: 0

strategy:
  type: round_robin

targets:
  - id: api_1
    address: "localhost:9001"
    weight: 1

  - id: api_2
    address: "localhost:9002"
    weight: 1

health:
  enabled: true
  interval: "10s"
  timeout: "5s"
  mode: tcp
  path: "/health"

metrics:
  enabled: true
  address: ":9090"

logging:
  level: info
  format: text

retry_attempts: 1
```

## Configuration Reference

### Server

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `address` | string | `:42069` | Listener address |
| `read_header_timeout` | duration | `5s` | Maximum time to read request headers |
| `read_timeout` | duration | `10s` | Maximum time to read the full request |
| `write_timeout` | duration | `10s` | Maximum time to write the response |
| `idle_timeout` | duration | `60s` | Maximum idle keep-alive time |
| `shutdown_timeout` | duration | `5s` | Graceful shutdown deadline |
| `max_inflight` | integer | `0` | Global concurrent request limit. `0` disables the limit. |
| `rate_limit_rps` | integer | `0` | Per-IP requests per second. `0` disables rate limiting. |

The rate limiter uses a token bucket per client IP. The burst size equals `rate_limit_rps`. Idle buckets are swept periodically. Relay uses the direct peer address and does not trust `X-Forwarded-For`.

### Strategy

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `type` | string | `round_robin` | Balancing algorithm |

Supported values:

- `round_robin`
- `weighted_round_robin`
- `least_conn`
- `least_response_time`
- `consistent_hash`

### Targets

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `id` | string | required | Unique target identifier |
| `address` | string | required | Backend `host:port` |
| `weight` | integer | `1` | Relative weight for `weighted_round_robin` |

Only `weighted_round_robin` uses `weight`. The other strategies accept it but ignore it.

After a reload, Relay asks the active strategy to prune state for deleted targets.

### Health

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | boolean | `false` | Enable active health checks |
| `interval` | duration | `0` | Time between health passes |
| `timeout` | duration | `0` | Timeout for one probe |
| `mode` | string | `tcp` | Probe type: `tcp` or `http` |
| `path` | string | `/health` | HTTP probe path |

The checker runs one pass immediately at startup, then repeats on the interval. A target becomes healthy after a successful probe and unhealthy after a failed probe.

When active checks are disabled, Relay marks all targets healthy at startup and uses passive tracking:

- Three consecutive transport or `5xx` failures mark a target unhealthy.
- A successful request resets the failure count and marks the target healthy again.

### Metrics

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | boolean | `false` | Start the metrics listener |
| `address` | string | empty | Metrics listener address |

When enabled, Relay serves Prometheus metrics on a separate HTTP listener. Example scrape target:

```yaml
scrape_configs:
  - job_name: relay
    static_configs:
      - targets: ["relay:9090"]
```

Available metrics:

| Metric | Type | Labels | Description |
| --- | --- | --- | --- |
| `relay_requests_total` | counter | none | Requests handled |
| `relay_request_errors_total` | counter | none | Failed or rejected requests |
| `relay_healthy_targets` | gauge | none | Current healthy target count |
| `relay_target_requests_total` | counter | `target`, `code` | Requests routed to each target |
| `relay_request_duration_seconds` | histogram | none | End-to-end request latency |

### Logging

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `level` | string | `info` | `debug`, `info`, `warn`, `error`, or `off` |
| `format` | string | `text` | `text` or `json` |

Use `level: off` for quiet test runs. Use `format: json` when shipping logs to an aggregator.

### Retries

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `retry_attempts` | integer | `1` | Extra attempts for `GET` and `HEAD` requests |

Rules:

- A missing value means `1`.
- `0` disables retries.
- Negative values are rejected.
- The total attempt count is `retry_attempts + 1`.
- Relay retries only `GET` and `HEAD`.
- Relay retries only transport failures.
- Relay does not retry HTTP `5xx` responses.
- Relay excludes already attempted targets when an alternative is available.
- If every healthy target was attempted, Relay can select from the full healthy set again.

## Strategy Guide

| Strategy | Selection | Best for |
| --- | --- | --- |
| `round_robin` | Equal rotation across healthy targets | Homogeneous backends |
| `weighted_round_robin` | Smooth rotation by weight | Backends with different capacity |
| `least_conn` | Fewest in-flight requests | Uneven request durations |
| `least_response_time` | Lowest EWMA response time | Latency-sensitive traffic |
| `consistent_hash` | Hash key onto a virtual-node ring | Sticky sessions and cache affinity |

`consistent_hash` uses `X-Request-ID` as the hash key when present. Otherwise it uses the client IP. Relay sets a generated `X-Request-ID` when the client did not send one. The generated value does not replace the client IP as the hash key.

Use the same `X-Request-ID` to pin a client to a target:

```bash
curl -H "X-Request-ID: alice" http://localhost:8080/
```

Without an explicit ID, clients behind a NAT can share the same client IP and therefore the same hash key.

## Operational Endpoints

These endpoints are served by the main HTTP listener.

| Endpoint | Description |
| --- | --- |
| `/healthz` | Liveness. Returns `200 ok` while the process runs. |
| `/readyz` | Readiness. Returns `200 ready` when at least one target is healthy. Otherwise returns `503`. |
| `/admin/targets` | Target state as JSON |
| All other paths | Forwarded to a backend target |

With active health checks enabled, `/readyz` can return `503` briefly at startup until the first health pass completes.

Example admin response:

```json
[
  {
    "id": "api_1",
    "address": "localhost:9001",
    "weight": 1,
    "healthy": true,
    "consecutive_failures": 0,
    "last_checked": "2026-11-07T12:34:56Z"
  }
]
```

`last_checked` is an empty string until the first health check. The admin and probe endpoints have no authentication. Restrict access at the network or ingress layer when Relay is exposed to untrusted clients.

## HTTP Status Behavior

| Status | Condition |
| --- | --- |
| `429` | Rate limit or in-flight limit exceeded |
| `502` | Backend transport failure on the final attempt, or invalid target URL |
| `503` | No healthy targets available |
| `504` | Backend timeout on the final attempt |

Relay passes other backend responses through unchanged.

## Configuration Reload

Send `SIGHUP` to reload the active config file:

```bash
kill -HUP <relay-pid>
```

Find the PID if needed:

```bash
pgrep -f './bin/relay'
```

These changes apply without a restart:

| Setting | Behavior |
| --- | --- |
| `targets` | Add, update, remove, and reorder targets |
| `strategy.type` | Swap the active strategy |
| `health.*` | Stop the old checker and rebuild it |
| `max_inflight` | Replace the in-flight limit |
| `rate_limit_rps` | Replace the rate limiter |
| `retry_attempts` | Replace the retry count |

These changes require a restart:

| Setting | Why |
| --- | --- |
| `server.address` | The listener is already bound |
| `server.read_header_timeout` | HTTP server timeout is set at construction |
| `server.read_timeout` | HTTP server timeout is set at construction |
| `server.write_timeout` | HTTP server timeout is set at construction |
| `server.idle_timeout` | HTTP server timeout is set at construction |
| `server.shutdown_timeout` | The CLI shutdown context uses the startup value |
| `metrics.enabled` | The metrics listener is started at boot |
| `metrics.address` | The metrics listener is already bound |
| `logging.level` | The logger is constructed at boot |
| `logging.format` | The logger is constructed at boot |

Relay logs a warning when a restart-only field changes.

Reload behavior details:

- An invalid file or validation error keeps the old configuration active.
- A target address change resets that target's health and failure state.
- A target weight change keeps the current health state.
- A removed target is pruned from the active strategy where the strategy supports pruning.
- If health settings change, Relay cancels and waits for the old health pass before it rebuilds the checker.
- If only one limit changes, Relay replaces only that part. It preserves the other limiter state.

## Common Recipes

### Minimal config

```yaml
targets:
  - id: api_1
    address: "localhost:9001"
```

This uses the default server address `:42069`, round robin, TCP health checks disabled, metrics disabled, and text logging at `info`.

### HTTP health checks

```yaml
health:
  enabled: true
  interval: "5s"
  timeout: "2s"
  mode: http
  path: "/health"
```

Relay accepts a target when the probe returns any `2xx` status.

### Disable retries

```yaml
retry_attempts: 0
```

### Rate limit and concurrency limit

```yaml
server:
  address: ":8080"
  max_inflight: 200
  rate_limit_rps: 20
```

`max_inflight` is a global limit. `rate_limit_rps` is a per-client-IP limit.

### Sticky routing

```yaml
strategy:
  type: consistent_hash
```

```bash
curl -H "X-Request-ID: user-42" http://localhost:8080/
```

### Passive health only

```yaml
health:
  enabled: false
```

All targets start healthy. Three consecutive failures mark a target unhealthy. A success recovers it.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| `503 no available targets` | Inspect `/admin/targets`. Check target health and probe settings. |
| `429 Too Many Requests` | Lower traffic, raise `rate_limit_rps`, or raise `max_inflight`. |
| `502 Bad Gateway` | Check that the target address is reachable over HTTP. |
| `504 Gateway Timeout` | Check backend latency and upstream timeouts. |
| Config fails to load | Unknown fields are rejected. Check the field names and types. |
| Health checks never pass | Check the target address, health mode, timeout, and path. |
| Metrics listener is unreachable | Set `metrics.enabled: true` and use an unused `metrics.address`. |
| Reload does not change a field | Check the log for a restart-required warning. |

## Tests and Load Checks

```bash
make test
make test-race
make lint
make all
```

The repository also includes k6 scripts:

- `examples/load/k6.js` for baseline throughput
- `examples/load/strategies.js` for backend distribution across strategies

The chaos test in `examples/chaos` kills a backend during concurrent requests and checks that Relay fails over without returning `500`.
