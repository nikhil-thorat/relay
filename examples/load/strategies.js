import http from "k6/http";
import { check } from "k6";
import { Counter } from "k6/metrics";

// Strategy lab traffic: sends requests through relay (localhost:8080)
// and tallies which backend answered, so distribution per strategy is visible.
//
// Usage (from repo root, relay stack already up):
//   k6 run examples/load/strategies.js
// Switch strategy without touching backends:
//   RELAY_CONFIG=/configs/wrr.yml docker compose -f examples/docker-compose.yml up -d relay
//
// Sticky check for consistent_hash (run separately, plain curl):
//   for i in 1 2 3 4 5; do curl -s -H "X-Request-ID: alice" localhost:8080/; done
//   # same backend every time; change to "bob" -> (usually) another backend.

const api1 = new Counter("backend_api_1");
const api2 = new Counter("backend_api_2");
const api3 = new Counter("backend_api_3");

export const options = {
  vus: 30,
  duration: "20s",
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(99)<100"],
  },
};

export default function () {
  const res = http.get("http://localhost:8080/");
  check(res, { "status 200": (r) => r.status === 200 });

  const body = res.body || "";
  if (body.includes("api_1")) {
    api1.add(1);
  } else if (body.includes("api_2")) {
    api2.add(1);
  } else if (body.includes("api_3")) {
    api3.add(1);
  }
}
