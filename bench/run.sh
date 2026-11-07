#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE=(docker compose -f bench/docker-compose.yml)
RESULTS="bench/results/latest"
RAW="$RESULTS/raw"
MANIFEST="$RESULTS/manifest.csv"
NETWORK="relay-bench-net"
K6_IMAGE="grafana/k6@sha256:3e4186aea62e11a45560bfc51e19eeafe82fc07ce069c78aff44b7ff893d9a5e"

REPS="${REPS:-3}"
DURATION="${DURATION:-60s}"
WARMUP="${WARMUP:-10s}"
PRE_VUS="${PRE_VUS:-200}"
MAX_VUS="${MAX_VUS:-3000}"
CH_REPEATS="${CH_REPEATS:-10}"
SAT_DURATION="${SAT_DURATION:-20s}"
SATURATION_RATES="${SATURATION_RATES:-500 1000 2000 4000 8000 12000 16000 24000 32000 48000}"
BENCH_RATE="${BENCH_RATE:-}"

log() {
  printf '\n== %s ==\n' "$*"
}

wait_ready() {
  local i
  for i in $(seq 1 60); do
    if curl -fsS http://localhost:8080/readyz >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  printf 'relay did not become ready\n' >&2
  return 1
}

check_environment() {
  local ac governor
  ac="$(cat /sys/class/power_supply/AC*/online 2>/dev/null | head -1 || true)"
  governor="$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || true)"
  if [[ "$ac" != "1" ]]; then
    printf 'warning: AC power is not online\n' >&2
  fi
  if [[ "$governor" != "performance" ]]; then
    printf 'warning: CPU governor is %s\n' "$governor" >&2
  fi
}

write_metadata() {
  {
    printf 'date=%s\n' "$(date -Is)"
    printf 'commit=%s\n' "$(git rev-parse HEAD)"
    printf 'host=%s\n' "$(cat /sys/devices/virtual/dmi/id/product_name 2>/dev/null || true)"
    printf 'cpu=%s\n' "$(lscpu | awk -F: '/Model name/ {gsub(/^[ \t]+/, "", $2); print $2; exit}')"
    printf 'cores=%s\n' "$(nproc)"
    printf 'memory_kib=%s\n' "$(awk '/MemTotal/ {print $2}' /proc/meminfo)"
    printf 'os=%s\n' "$(. /etc/os-release && printf '%s' "$PRETTY_NAME")"
    printf 'kernel=%s\n' "$(uname -r)"
    printf 'go=%s\n' "$(go version)"
    printf 'docker=%s\n' "$(docker --version)"
    printf 'compose=%s\n' "$(docker compose version --short)"
    printf 'k6=%s\n' "$(docker run --rm "$K6_IMAGE" version | head -1)"
    printf 'governor=%s\n' "$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || true)"
    printf 'ac_online=%s\n' "$(cat /sys/class/power_supply/AC*/online 2>/dev/null | paste -sd, - || true)"
    printf 'battery_status=%s\n' "$(cat /sys/class/power_supply/BAT*/status 2>/dev/null | paste -sd, - || true)"
    printf 'battery_capacity=%s\n' "$(cat /sys/class/power_supply/BAT*/capacity 2>/dev/null | paste -sd, - || true)"
  } >"$RESULTS/metadata.txt"
}

prepare() {
  rm -rf "$RESULTS"
  mkdir -p "$RAW"
  printf 'file,suite,scenario,rate,duration,rep\n' >"$MANIFEST"
  write_metadata
}

compose_up() {
  "${COMPOSE[@]}" build
  "${COMPOSE[@]}" up -d api_1 api_2 api_3 relay
  wait_ready
}

compose_down() {
  "${COMPOSE[@]}" down -v --remove-orphans
}

switch_config() {
  RELAY_CONFIG="/configs/$1" "${COMPOSE[@]}" up -d --force-recreate relay
  wait_ready
}

record() {
  local file="$1" suite="$2" scenario="$3" rate="$4" duration="$5" rep="$6"
  printf '%s,%s,%s,%s,%s,%s\n' "$file" "$suite" "$scenario" "$rate" "$duration" "$rep" >>"$MANIFEST"
}

run_k6() {
  local suite="$1" scenario="$2" rate="$3" duration="$4" rep="$5" script="$6" base="$7"
  local file="${suite}__${scenario}__${rate}__${rep}.json"
  "${COMPOSE[@]}" --profile load run --rm --no-deps \
    -e BASE_URL="$base" \
    -e RATE="$rate" \
    -e DURATION="$duration" \
    -e PRE_VUS="$PRE_VUS" \
    -e MAX_VUS="$MAX_VUS" \
    -e KEYS="${KEYS:-100}" \
    -e REPEATS="${CH_REPEATS:-10}" \
    k6 run --quiet --summary-export "/results/latest/raw/$file" "/scripts/$script" >/dev/null
  if [[ ! -f "$RAW/$file" ]]; then
    printf 'missing k6 summary: %s\n' "$file" >&2
    return 1
  fi
  record "$file" "$suite" "$scenario" "$rate" "$duration" "$rep"
}

warmup_k6() {
  local rate="$1" duration="$2" script="$3" base="$4"
  "${COMPOSE[@]}" --profile load run --rm --no-deps \
    -e BASE_URL="$base" \
    -e RATE="$rate" \
    -e DURATION="$duration" \
    -e PRE_VUS="$PRE_VUS" \
    -e MAX_VUS="$MAX_VUS" \
    k6 run --quiet "/scripts/$script" >/dev/null
}

passes() {
  local file="$1"
  python3 - "$RAW/$file" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1]))["metrics"]
p99 = data.get("http_req_duration", {}).get("p(99)", 1e9)
failed = data.get("http_req_failed", {})
if "value" in failed:
    errors = failed["value"]
elif "rate" in failed:
    errors = failed["rate"]
else:
    total = failed.get("passes", 0) + failed.get("fails", 0)
    errors = failed.get("fails", 0) / total if total else 0
dropped = data.get("dropped_iterations", {}).get("count", 0)
raise SystemExit(0 if p99 < 100 and errors < 0.001 and dropped == 0 else 1)
PY
}

calc_rate() {
  local best="$1"
  python3 - "$best" <<'PY'
import sys

best = float(sys.argv[1])
print(max(100, int(best * 0.7 / 100) * 100))
PY
}

saturation() {
  log "saturation"
  switch_config rr.yml
  local rate best_relay=0 best_direct=0
  for rate in $SATURATION_RATES; do
    run_k6 saturation saturation_direct "$rate" "$SAT_DURATION" 1 saturation.js http://api_1:9001
    if passes "saturation__saturation_direct__${rate}__1.json"; then
      best_direct="$rate"
    fi

    run_k6 saturation saturation_relay "$rate" "$SAT_DURATION" 1 saturation.js http://relay:8080
    if passes "saturation__saturation_relay__${rate}__1.json"; then
      best_relay="$rate"
    elif (( best_relay > 0 )); then
      printf 'rate=%s direct_best=%s relay_best=%s\n' "$rate" "$best_direct" "$best_relay"
      break
    fi
    printf 'rate=%s direct_best=%s relay_best=%s\n' "$rate" "$best_direct" "$best_relay"
  done
  printf 'best_relay=%s\nbest_direct=%s\n' "$best_relay" "$best_direct" >"$RESULTS/saturation.env"
}

overhead() {
  log "overhead"
  local rate="$1" rep
  warmup_k6 "$rate" "$WARMUP" overhead.js http://api_1:9001
  for rep in $(seq 1 "$REPS"); do
    run_k6 overhead direct "$rate" "$DURATION" "$rep" overhead.js http://api_1:9001
  done
}

run_strategy() {
  local scenario="$1" config="$2" rate="$3" rep
  switch_config "$config"
  warmup_k6 "$rate" "$WARMUP" strategies.js http://relay:8080
  for rep in $(seq 1 "$REPS"); do
    run_k6 strategies "$scenario" "$rate" "$DURATION" "$rep" strategies.js http://relay:8080
  done
}

strategies() {
  log "strategies"
  local rate="$1"
  run_strategy round_robin rr.yml "$rate"
  run_strategy weighted_round_robin wrr.yml "$rate"
  run_strategy least_conn least-conn.yml "$rate"
  run_strategy least_response_time lrt.yml "$rate"
  run_strategy consistent_hash consistent-hash.yml "$rate"
}

health() {
  log "health"
  local rate="$1" rep
  switch_config health-off.yml
  warmup_k6 "$rate" "$WARMUP" overhead.js http://relay:8080
  for rep in $(seq 1 "$REPS"); do
    run_k6 health health_off "$rate" "$DURATION" "$rep" overhead.js http://relay:8080
  done
}

sticky() {
  log "consistent hash sticky"
  switch_config consistent-hash.yml
  run_k6 behavior sticky 0 "$DURATION" 1 ch-sticky.js http://relay:8080
}

slow_backend() {
  local scenario="$1" config="$2" rate="$3"
  API1_DELAY=20ms "${COMPOSE[@]}" up -d --force-recreate api_1 >/dev/null
  API1_DELAY=20ms switch_config "$config"
  warmup_k6 "$rate" 15s strategies.js http://relay:8080
  run_k6 behavior "${scenario}_slow" "$rate" "$DURATION" 1 strategies.js http://relay:8080
}

behavior() {
  log "behavior"
  local rate
  rate="$(python3 - "$1" <<'PY'
import sys

rate = float(sys.argv[1])
print(max(100, int(rate * 0.3 / 100) * 100))
PY
)"
  slow_backend least_conn least-conn.yml "$rate"
  slow_backend least_response_time lrt.yml "$rate"
  API1_DELAY=0ms "${COMPOSE[@]}" up -d --force-recreate api_1 >/dev/null
  wait_ready
  sticky
}

start_bg_k6() {
  local name="$1" file="$2" script="$3" base="$4" rate="$5" duration="$6"
  docker rm -f "$name" >/dev/null 2>&1 || true
  docker run -d --name "$name" \
    --network "$NETWORK" \
    --cpuset-cpus "8-11" \
    --user 0:0 \
    --ulimit nofile=65535:65535 \
    --log-driver none \
    -e GOMAXPROCS=4 \
    -e BASE_URL="$base" \
    -e RATE="$rate" \
    -e DURATION="$duration" \
    -e PRE_VUS="$PRE_VUS" \
    -e MAX_VUS="$MAX_VUS" \
    -v "$ROOT/bench/k6:/scripts:ro,z" \
    -v "$ROOT/bench/results:/results:z" \
    "$K6_IMAGE" run --quiet --summary-export "/results/latest/raw/$file" "/scripts/$script" >/dev/null
  printf '%s' "$name"
}

wait_bg_k6() {
  local name="$1" file="$2" suite="$3" scenario="$4" rate="$5" duration="$6"
  local code
  code="$(docker wait "$name")"
  docker rm "$name" >/dev/null
  if [[ "$code" != "0" ]]; then
    printf 'k6 %s exited with %s\n' "$name" "$code" >&2
  fi
  record "$file" "$suite" "$scenario" "$rate" "$duration" 1
}

operations() {
  log "operations"
  local rate="$1"
  local file name

  switch_config rr.yml
  file="operations__reload__${rate}__1.json"
  name="$(start_bg_k6 relay-bench-k6-reload "$file" overhead.js http://relay:8080 "$rate" "$DURATION")"
  sleep 15
  "${COMPOSE[@]}" kill -s HUP relay
  sleep 20
  "${COMPOSE[@]}" kill -s HUP relay
  wait_bg_k6 "$name" "$file" operations reload "$rate" "$DURATION"

  switch_config rr.yml
  file="operations__failover__${rate}__1.json"
  name="$(start_bg_k6 relay-bench-k6-failover "$file" overhead.js http://relay:8080 "$rate" "$DURATION")"
  sleep 15
  "${COMPOSE[@]}" stop api_2 >/dev/null
  sleep 20
  "${COMPOSE[@]}" start api_2 >/dev/null
  wait_bg_k6 "$name" "$file" operations failover "$rate" "$DURATION"
}

summarize() {
  python3 bench/summarize.py
}

smoke() {
  prepare
  compose_up
  BENCH_RATE="${BENCH_RATE:-500}"
  switch_config rr.yml
  run_k6 smoke smoke "$BENCH_RATE" 10s 1 overhead.js http://relay:8080
  summarize
  compose_down
}

all() {
  prepare
  check_environment
  compose_up
  saturation
  . "$RESULTS/saturation.env"
  if [[ -z "$BENCH_RATE" ]]; then
    BENCH_RATE="$(calc_rate "$best_relay")"
  fi
  printf 'bench_rate=%s\n' "$BENCH_RATE"
  overhead "$BENCH_RATE"
  strategies "$BENCH_RATE"
  health "$BENCH_RATE"
  behavior "$BENCH_RATE"
  operations "$BENCH_RATE"
  summarize
  compose_down
}

case "${1:-all}" in
  all) all ;;
  smoke) smoke ;;
  up) prepare; compose_up ;;
  down) compose_down ;;
  saturation) prepare; compose_up; saturation; summarize ;;
  overhead) prepare; compose_up; overhead "${BENCH_RATE:-1000}"; summarize ;;
  strategies) prepare; compose_up; strategies "${BENCH_RATE:-1000}"; summarize ;;
  health) prepare; compose_up; health "${BENCH_RATE:-1000}"; summarize ;;
  behavior) prepare; compose_up; behavior "${BENCH_RATE:-1000}"; summarize ;;
  operations) prepare; compose_up; operations "${BENCH_RATE:-1000}"; summarize ;;
  *)
    printf 'usage: %s [all|smoke|up|down|saturation|overhead|strategies|health|behavior|operations]\n' "$0" >&2
    exit 1
    ;;
esac
