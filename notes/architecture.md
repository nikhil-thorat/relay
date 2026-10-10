# Relay Architecture

```
client -> relay -> pool -> strategy -> proxy -> backend
              |-> health checker -> pool
              |-> metrics (prometheus)
```

- `cmd/relay/main.go`: flags -> load -> `relay.New` -> `Start` -> `Shutdown`. Handles `SIGHUP` reload.
- `internal/config`: YAML load, defaults, validation.
- `internal/target`: `Pool` (private maps, value copies, `Remove`, failure counters).
- `internal/strategy`: `round_robin`, `weighted_round_robin`, `least_conn`, `least_response_time`, `consistent_hash`.
- `internal/balancer`: `Next`, `NextForRequest` (sticky), `Before`/`After` hooks, `ReportFailure/Success`.
- `internal/proxy`: shared `Transport`, `ErrorHandler` 502/504, retry GET/HEAD once, 429 rate/max-inflight, `X-Forwarded-For`/`X-Request-ID`.
- `internal/health`: TCP + HTTP `/health` checks, concurrent, gauge updates.
- `internal/metrics`: requests, errors, healthy gauge, per-target+code, duration histogram.
- `internal/relay`: timeouts, graceful drain, `Reload`, `/admin/targets`, `/healthz`, `/readyz`.

## Demo script

```bash
go run ./examples/backends -port 9001 -id api_1 &
go run ./examples/backends -port 9002 -id api_2 &
go run ./cmd/relay -config relay.yml
curl localhost:8080/
curl localhost:8080/healthz
curl localhost:8080/admin/targets
kill %1 # traffic continues via retry
k6 run --vus 50 --duration 30s examples/load/k6.js
docker compose -f examples/docker-compose.yml up
go test -race ./...
```
