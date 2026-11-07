# Benchmarks

These numbers come from a single laptop. They show the cost of the Relay proxy path and the behavior of each balancing strategy on a controlled local stack. They are not a multi-host capacity claim.

## Test setup

| Item | Value |
| --- | --- |
| Host | ASUS Vivobook M6500QC |
| CPU | AMD Ryzen 5 5600H, 6 cores / 12 threads |
| Memory | 15 GiB |
| OS | Fedora Linux 44, kernel 7.2.15 |
| Go | 1.26.10 |
| Docker / Compose | 29.9.2 / 5.6.0 |
| Load generator | k6 v2.3.0 (`grafana/k6` container) |
| Power profile | AC power, `performance` governor |

Three backends and Relay run in Docker on a bridge network. Each container is pinned to separate CPUs with `GOMAXPROCS=4`:

| Role | CPUs |
| --- | --- |
| Relay | `0-3` |
| Backends | `4-7` |
| k6 | `8-11` |

All requests are HTTP `GET` with a 1 KiB response body. The load generator uses a constant arrival rate for 60 seconds. The main suite uses 5600 requests per second, which is about 70 percent of the measured saturation point. Results are the median of 3 runs. Saturation, behavior, and operations rows are single runs.

Saturation is measured in 20-second steps up to 48000 offered requests per second. A step passes when it has no errors, no dropped iterations, and completed requests stay near the offered rate.

## Reproduce

```bash
make bench        # full load suite, writes bench/results/latest
make bench-smoke  # short validation run
make bench-micro  # Go microbenchmarks
```

The full suite starts the Docker stack, builds the binaries, runs saturation and load tests, and writes `summary.csv` and `summary.md`. Raw k6 JSON stays under `bench/results/latest/raw/` and is not committed.

Microbenchmarks use:

```bash
go test ./internal/balancer ./internal/proxy -run '^$' -bench . -benchmem -count=5 -timeout 0
```

## Proxy overhead

Measured at 5600 requests per second with round-robin routing:

| Path | RPS | p50 | p95 | p99 | Errors |
| --- | ---: | ---: | ---: | ---: | ---: |
| Direct to backends | 5599.97 | 0.11 ms | 0.17 ms | 0.26 ms | 0 |
| Relay, health checks on | 5599.97 | 0.22 ms | 0.35 ms | 0.68 ms | 0 |
| Relay, health checks off | 5599.98 | 0.22 ms | 0.37 ms | 0.71 ms | 0 |

Relay added about 0.11 ms at p50 and 0.42 ms at p99. Disabling active health checks made no measurable difference at this load.

## Strategies

Weighted targets are `3:2:1` for `api_1`, `api_2`, and `api_3`. All runs used 5600 requests per second.

| Strategy | RPS | p50 | p95 | p99 | Errors | api_1 | api_2 | api_3 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `round_robin` | 5599.97 | 0.22 ms | 0.35 ms | 0.68 ms | 0 | 33.33% | 33.33% | 33.33% |
| `weighted_round_robin` | 5599.98 | 0.22 ms | 0.35 ms | 0.67 ms | 0 | 50.00% | 33.33% | 16.67% |
| `least_conn` | 5599.98 | 0.22 ms | 0.35 ms | 0.70 ms | 0 | 33.34% | 33.33% | 33.33% |
| `least_response_time` | 5599.97 | 0.22 ms | 0.34 ms | 0.68 ms | 0 | 0% | 0% | 100% |
| `consistent_hash` | 5599.98 | 0.23 ms | 0.42 ms | 0.93 ms | 0 | 38.37% | 31.32% | 30.31% |

Weighted routing matched the configured `3:2:1` ratio exactly. Least connections split evenly because the three backends had equal latency and no queued work. Consistent hashing spread keys across all targets; a separate sticky test sent 100 keys 10 times each and recorded 0 target changes.

`least_response_time` converged to a single target. With equal backend latency, the current strategy always selects the lowest EWMA and does not explore other targets, so it stays on the first target that becomes lowest. Against a slow target it moves traffic away correctly, but under equal latency it does not balance. This is a known tradeoff of the current implementation.

## Behavior and control-plane events

| Scenario | Result |
| --- | --- |
| `least_conn` with `api_1` delayed 20 ms | `api_1` 3.06%, `api_2` 55.07%, `api_3` 41.87%; p99 20.41 ms; 0 errors |
| `least_response_time` with `api_1` delayed 20 ms | `api_1` 0%, `api_3` 100%; p99 0.55 ms; 0 errors |
| Consistent-hash stickiness | 100 keys x 10 requests, 0 target changes |
| `SIGHUP` reload during load | 5599.97 RPS, p99 0.65 ms, 0 errors |
| Backend stop and restart during load | 5599.98 RPS, p99 0.65 ms, 0 errors |

Least connections moved traffic away from the delayed backend and away from an increasing in-flight count. Least response time also avoided the slow backend, but it still concentrated all traffic on one target.

## Saturation

| Offered RPS | Relay RPS | p99 | Dropped |
| --- | ---: | ---: | ---: |
| 4000 | 3999.94 | 0.58 ms | 0 |
| 8000 | 7999.82 | 0.93 ms | 0 |
| 12000 | 11990.18 | 6.92 ms | 193 |

Relay held 8000 requests per second with p99 under 1 ms and no dropped iterations. At 12000 offered requests per second it dropped iterations and p99 rose to 6.92 ms. For comparison, direct-to-backend traffic at 8000 offered requests per second achieved 7998.65 RPS with p99 0.43 ms and 25 dropped iterations. Both paths saturate near the same point because k6 shares the same host, so the test measures Relay overhead more reliably than absolute capacity.

## Microbenchmarks

Strategy selection, median of 5 runs:

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkBalancerNext/round_robin` | 153.2 | 128 | 1 |
| `BenchmarkBalancerNext/weighted_round_robin` | 272.3 | 128 | 1 |
| `BenchmarkBalancerNext/least_conn` | 189.3 | 128 | 1 |
| `BenchmarkBalancerNext/least_response_time` | 301.2 | 416 | 4 |
| `BenchmarkBalancerNext/consistent_hash` | 319.0 | 376 | 5 |
| `BenchmarkBalancerSelectForRequest/round_robin` | 239.6 | 256 | 2 |
| `BenchmarkBalancerSelectForRequest/weighted_round_robin` | 317.9 | 256 | 2 |
| `BenchmarkBalancerSelectForRequest/least_conn` | 272.8 | 256 | 2 |
| `BenchmarkBalancerSelectForRequest/least_response_time` | 333.0 | 384 | 4 |
| `BenchmarkBalancerSelectForRequest/consistent_hash` | 301.0 | 264 | 3 |
| `BenchmarkProxy` | 91382 | 45720 | 104 |

`BenchmarkProxy` measures one request through the proxy with an in-process backend.

## Limits

- Results come from one laptop with k6, Relay, and the backends on the same CPU package. Use a separate load generator for capacity planning.
- Docker bridge networking adds overhead that a host-network or bare-metal test would not show.
- The host ran on AC power with the `performance` governor and no other workloads.
- Absolute throughput depends on backend work, response size, connection reuse, and kernel settings. The relative overhead and strategy behavior are the useful results.
