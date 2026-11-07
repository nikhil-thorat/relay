import { Counter } from "k6/metrics";
import { arrivalOptions, get } from "./common.js";

const api1 = new Counter("backend_api_1");
const api2 = new Counter("backend_api_2");
const api3 = new Counter("backend_api_3");

export const options = {
  ...arrivalOptions,
  discardResponseBodies: false,
};

export default function () {
  const res = get("/", { "X-Request-ID": `${__VU}-${__ITER}` });
  if (res.status !== 200) {
    return;
  }

  const body = res.body || "";
  if (body.includes("api_1")) {
    api1.add(1);
  } else if (body.includes("api_2")) {
    api2.add(1);
  } else if (body.includes("api_3")) {
    api3.add(1);
  }
}
