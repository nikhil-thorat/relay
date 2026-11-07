import http from "k6/http";

export const BASE_URL = __ENV.BASE_URL || "http://relay:8080";
export const RATE = Number(__ENV.RATE || 1000);
export const DURATION = __ENV.DURATION || "20s";
export const PRE_VUS = Number(__ENV.PRE_VUS || 200);
export const MAX_VUS = Number(__ENV.MAX_VUS || 3000);

export const arrivalOptions = {
  discardResponseBodies: true,
  summaryTrendStats: ["avg", "min", "med", "max", "p(90)", "p(95)", "p(99)"],
  scenarios: {
    load: {
      executor: "constant-arrival-rate",
      rate: RATE,
      timeUnit: "1s",
      duration: DURATION,
      preAllocatedVUs: PRE_VUS,
      maxVUs: MAX_VUS,
      gracefulStop: "10s",
    },
  },
};

export function get(path = "/", headers = {}) {
  return http.get(`${BASE_URL}${path}`, { headers });
}
