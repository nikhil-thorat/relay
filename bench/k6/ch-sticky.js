import { check } from "k6";
import { Counter } from "k6/metrics";
import { get } from "./common.js";

const mismatches = new Counter("ch_mismatches");
const keys = Number(__ENV.KEYS || 100);
const repeats = Number(__ENV.REPEATS || 10);
const seen = Object.create(null);

export const options = {
  summaryTrendStats: ["avg", "min", "med", "max", "p(90)", "p(95)", "p(99)"],
  scenarios: {
    sticky: {
      executor: "per-vu-iterations",
      vus: 1,
      iterations: keys * repeats,
      maxDuration: "2m",
    },
  },
};

export default function () {
  const key = `key-${__ITER % keys}`;
  const res = get("/", { "X-Request-ID": key });
  const target = (res.body || "").trim().split(/\s+/)[2] || "unknown";

  const ok = check(res, { "status 200": (r) => r.status === 200 });
  if (!ok) {
    return;
  }

  if (seen[key] === undefined) {
    seen[key] = target;
    return;
  }

  const same = seen[key] === target;
  check(res, { "sticky target": () => same });
  if (!same) {
    mismatches.add(1);
  }
}
