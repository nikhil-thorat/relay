# Relay Roadmap — Phased Plan

Goal: make Relay interview-ready — correct, robust, observable, extensible.
Order is intentional. Do not skip phases.

---

## 0. Ground Rules (applies to every phase)

### Coding style — how this repo writes code

Keep it simple, clean, easy to understand. Match existing style:

- One concern per package under `internal/`: `config, target, strategy, balancer, proxy, health, metrics, logging, relay`.
- Plain structs + small `New(...)` constructor. No globals, no init magic.
- Small methods, early returns. Stdlib first (`net/http`, `sync`, `slog`, `flag`). No web frameworks.
- Explicit errors with `errors.New / fmt.Errorf`. Wrap with context: `fmt.Errorf("load config %s: %w", path, err)`.
- `slog` key-value logging with `component` field via `logger.WithComponent("proxy")`. Keep messages lowercase.
- YAML tags only in `config`. Keep `Weight int`, `time.Duration` conventions.
- No clever generics, reflection, or over-abstraction. If a file grows past ~150 lines, split it.
- Keep `cmd/relay/main.go` thin: flags -> load -> `relay.New` -> `Start` -> `Shutdown`.

### Testing contract — every package tested for validity

- Stdlib `testing` only. Table-driven tests where possible.
- Naming: `TestNew, TestAdd, TestSelect, TestNext, TestServeHTTP..., TestCheck..., TestLoad...`.
- Isolation: fresh `prometheus.NewRegistry()` per test (see `proxy_test.go`, `relay_test.go`). Use `httptest.NewServer` for backends. Never bind fixed ports except `testdata/smoke.yml`.
- Every new feature needs:
  1. Unit test for happy path,
  2. Unit test for error/empty path,
  3. Concurrency test if shared state (`-race`, parallel goroutines),
  4. Integration touch in `internal/relay/*_test.go` or `smoke_test.go` if wiring changes.
- Run before each phase done: `make lint && go test -race ./...`.
- Update `examples/config/explicit.yml` if config schema changes. Add `invalid` fixture if new validation added.

---

## Phase 1 — Correctness and Thread-Safety

**Goal:** remove races and panics. Biggest interview ROI.

Tasks:
1. `strategy/round_robin.go`: protect `current` with `sync.Mutex`. Guard index: `rr.current % len(targets)` on entry. Keep single-target fast path.
2. `target/pool.go`: make `Targets, States, Order` private (`targets, states, order`). Fix `Get/List/Healthy` to return copies (`Target` value, not `*Target`). Add nil guard in `Add`. Add `Remove(id)` for completeness.
3. `strategy/factory.go`: fix typo `unkown -> unknown`. Keep `round_robin` string contract.
4. `relay/relay.go:127`: log metrics-server error instead of swallowing TODO.

Tests:
- `target/pool_test.go`: add `TestConcurrentAddGetHealthy`, `TestGetReturnsCopy`, `TestRemove`.
- `strategy/round_robin_test.go`: add `TestConcurrentSelect`, `TestShrinkingPoolNoPanic` (3 -> 2 -> 1 healthy).
- `balancer/balancer_test.go`: add `TestNextConcurrent`.
- Acceptance: `go test -race ./...` clean.

## Phase 2 — Robust Request Handling

**Goal:** proper proxy behavior, timeouts, status codes.

Tasks:
1. `proxy/proxy.go`: reuse one `httputil.ReverseProxy` with shared `http.Transport` (timeouts, keep-alive). Set `Proxy.ErrorHandler` to return `502/504` + `IncrementErrors`. Add `X-Forwarded-For`, `X-Request-ID` passthrough. Fix `slog` kv order (`"target", target.ID`).
2. `proxy/response_writer.go`: add `Write` override, forward `Flush/Hijack` (`http.Flusher`, `http.Hijacker`) so SSE/websockets do not break.
3. `relay/relay.go:75`: set `ReadHeaderTimeout: 5s, ReadTimeout, WriteTimeout, IdleTimeout` on both servers. Make shutdown timeout configurable (not hardcoded `5s` in `main.go`).
4. `config/validate.go`: validate `server.address` non-empty, `metrics.address` when enabled, `health.interval/timeout > 0` when enabled.
5. `cmd/relay/main.go`, `config/load.go`: wrap errors with file context.

Status code contract:
- `503` no healthy targets, `502` bad target URL / backend dial fail, `504` backend timeout.

Tests:
- `proxy/proxy_test.go`: `TestStatusCodes` (no-targets=503, bad-URL=502, backend-down=502), `TestHeadersForwarded`, `TestErrorHandlerIncrementsErrors`, `TestConcurrentProxy`.
- `config/validation_test.go + load_test.go`: new cases for bad addresses, zero health durations.
- `relay/runtime_test.go`: timeout config test.

## Phase 3 — Failure Handling: Health + Retries

**Goal:** survive backend failures.

Tasks:
1. `health/checker.go`: add HTTP mode (`GET /health`, expect 2xx, use `timeout`). Run checks concurrently (bounded goroutines), not sequentially. Keep TCP fallback via config `health.mode: tcp|http`. Keep transition logs `recovered / became unhealthy`.
2. Passive health: `proxy` reports backend 5xx/timeout to pool (evict after N fails). Keep it simple: counter in `TargetState{ConsecutiveFailures}`.
3. `balancer` + `proxy`: retry once on different backend for idempotent methods (`GET, HEAD`). No retry on `POST` by default. Config `retry_attempts: 1`.
4. `target/state.go`: extend to `{Healthy bool, ConsecutiveFailures int, LastChecked time.Time}`.

Tests:
- `health/checker_test.go`: `TestHTTPHealthy/Unhealthy`, `TestConcurrentRunDoesNotBlock`, `TestTransitionUpdatesGauge`.
- `proxy/proxy_test.go`: `TestRetryOnDifferentBackend`, `TestNoRetryOnPost`.
- `target/pool_test.go`: failure-counter behavior.
- Acceptance: kill one `examples/backends` mid-smoke, traffic continues.

## Phase 4 — Observability: Logging + Metrics

**Goal:** debuggable in production.

Tasks:
1. `logging/logger.go`: support `format: text|json` via config, keep `level: debug|info|warn|error|off`. Ensure `WithComponent` used everywhere.
2. `proxy/proxy.go`: log `request-id, method, path, target, status, duration` on every request. Log backend errors at `warn/error`.
3. `metrics/metrics.go`: add `relay_request_duration_seconds` histogram + `code` label on `target_requests`. Fix: backend 5xx increments `RequestErrorsTotal`. Keep `HealthyTargets` gauge updates in checker.
4. `cmd/relay/main.go`: log config path + listening addresses on start.

Tests:
- `metrics/metrics_test.go`: one test per new helper + label check via `testutil.ToFloat64`.
- `proxy/proxy_test.go`: assert duration metric observed, status captured.
- `logging`: `TestNewLevels`, `TestInvalidLevel`.

## Phase 5 — New Strategies

**Goal:** prove extensibility. `Weight` is currently dead — wire it.

Tasks (one file per strategy, same `Strategy` interface):
1. `weighted_round_robin.go`: smooth WRR using `Target.Weight`. Falls back to RR when weights equal/1.
2. `least_connections.go`: track in-flight per target (`atomic` counters inc/dec in `proxy`). Needs `Balancer` hook — keep simple: strategy holds `map[string]*atomic.Int64`.
3. `least_response_time.go` (your "LRT"): EWMA per target, pick lowest. Update on proxy completion.
4. `consistent_hash.go` / `ip_hash.go`: hash `RemoteAddr` or `X-Request-ID` for sticky routing.
5. Note: skip `LRU` — it is a cache eviction policy, not an LB strategy. Mention this in interviews.
6. `strategy/factory.go` + `config/validate.go`: allow `round_robin, weighted_round_robin, least_conn, least_response_time, consistent_hash`.

Tests (per strategy file):
- `TestSelectDistribution` (e.g. weights 2:1 over 300 picks), `TestEmptyReturnsError`, `TestConcurrentSelect`, `factory_test.go` new cases.

## Phase 6 — Lifecycle and Scalability

**Goal:** run as a real service.

Tasks:
1. Graceful drain: `Shutdown` cancels health, stops accepting, waits for in-flight (use `http.Server.Shutdown` correctly, make timeout configurable).
2. Dynamic reload: `SIGHUP` -> `config.Load` -> swap pool/strategy without dropping connections (keep v1 simple: log + rebuild pool).
3. Rate limiting (simple token bucket per IP) + `max_inflight` guard returning `429`.
4. Benchmarks: `BenchmarkBalancerNext`, `BenchmarkProxy`, `wrk/k6` script in `examples/load/`. Document RPS/p99.

Tests:
- `relay/runtime_test.go`: `TestShutdownDrainsInflight`, `TestReloadSwapsPool`.
- `Benchmark*` + `race` in CI. Add `Dockerfile` + `examples/docker-compose.yml` (2 backends + relay + prometheus).

## Phase 7 — Enchanting Features (interview wow)

Pick 2-3 only after Phase 1-4 done:

- `GET /admin/targets` (JSON pool health), `GET /healthz`, `GET /readyz`.
- Tiny dashboard: RPS, p99, healthy count (serve from metrics endpoint).
- OpenTelemetry tracing (trace-id through proxy).
- Chaos test: `examples/chaos/` kills backends randomly, asserts zero 500s to client (only 503 + retry).
- Docs: `notes/architecture.md` diagram (client -> relay -> pool -> strategy -> proxy) + demo video script using `examples/backends/main.go -port 9001/9002`.

---

## Execution Order Summary

`1 correctness -> 2 proxy -> 3 health/retries -> 4 observability -> 5 strategies -> 6 lifecycle/scale -> 7 wow`

Do not start Phase 5 before Phase 1 is green with `-race`.
