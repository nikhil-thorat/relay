import http from "k6/http";
import { check } from "k6";

export const options = {
  vus: 50,
  duration: "30s",
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(99)<100"],
  },
};

export default function () {
  const res = http.get("http://localhost:8080/");
  check(res, { "status 200": (r) => r.status === 200 });
}
