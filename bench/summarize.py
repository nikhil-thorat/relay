#!/usr/bin/env python3
import csv
import json
import statistics
from pathlib import Path

ROOT = Path(__file__).resolve().parent
RESULTS = ROOT / "results" / "latest"
MANIFEST = RESULTS / "manifest.csv"
SUMMARY = RESULTS / "summary.csv"
SUMMARY_MD = RESULTS / "summary.md"


def value(data, metric, key, default=0.0):
    return data.get("metrics", {}).get(metric, {}).get(key, default)


def error_rate(data):
    failed = data.get("metrics", {}).get("http_req_failed", {})
    if "value" in failed:
        return failed["value"]
    if "rate" in failed:
        return failed["rate"]
    total = failed.get("passes", 0) + failed.get("fails", 0)
    return failed.get("fails", 0) / total if total else 0.0


def check_rate(data):
    checks = data.get("metrics", {}).get("checks", {})
    if "value" in checks:
        return checks["value"]
    if "rate" in checks:
        return checks["rate"]
    total = checks.get("passes", 0) + checks.get("fails", 0)
    return checks.get("passes", 0) / total if total else 0.0


def median(values):
    values = [v for v in values if v is not None]
    return statistics.median(values) if values else 0.0


def read_runs():
    runs = []
    with MANIFEST.open(newline="") as f:
        for row in csv.DictReader(f):
            path = RESULTS / "raw" / row["file"]
            with path.open() as data_file:
                data = json.load(data_file)
            row["data"] = data
            runs.append(row)
    return runs


def group_runs(runs):
    groups = {}
    for run in runs:
        key = (run["suite"], run["scenario"], run["rate"])
        groups.setdefault(key, []).append(run)
    return groups


def summarize(groups):
    rows = []
    for (suite, scenario, rate), runs in sorted(groups.items()):
        api1 = median([value(r["data"], "backend_api_1", "count") for r in runs])
        api2 = median([value(r["data"], "backend_api_2", "count") for r in runs])
        api3 = median([value(r["data"], "backend_api_3", "count") for r in runs])
        total = api1 + api2 + api3
        rows.append(
            {
                "suite": suite,
                "scenario": scenario,
                "rate_rps": rate,
                "reps": len(runs),
                "duration": runs[0]["duration"],
                "throughput_rps": round(median([value(r["data"], "http_reqs", "rate") for r in runs]), 2),
                "p50_ms": round(median([value(r["data"], "http_req_duration", "med") for r in runs]), 2),
                "p95_ms": round(median([value(r["data"], "http_req_duration", "p(95)") for r in runs]), 2),
                "p99_ms": round(median([value(r["data"], "http_req_duration", "p(99)") for r in runs]), 2),
                "error_rate": round(median([error_rate(r["data"]) for r in runs]), 6),
                "dropped": int(median([value(r["data"], "dropped_iterations", "count") for r in runs])),
                "checks_pass": round(median([check_rate(r["data"]) for r in runs]), 6),
                "ch_mismatches": int(median([value(r["data"], "ch_mismatches", "count") for r in runs])),
                "api_1": int(api1),
                "api_2": int(api2),
                "api_3": int(api3),
                "api_1_pct": round(100 * api1 / total, 2) if total else 0.0,
                "api_2_pct": round(100 * api2 / total, 2) if total else 0.0,
                "api_3_pct": round(100 * api3 / total, 2) if total else 0.0,
            }
        )
    return rows


def write_csv(rows):
    if not rows:
        return
    with SUMMARY.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=list(rows[0].keys()))
        writer.writeheader()
        writer.writerows(rows)


def write_markdown(rows):
    lines = ["# Benchmark summary", ""]
    current = None
    for row in rows:
        if row["suite"] != current:
            current = row["suite"]
            lines.extend(
                [
                    f"## {current}",
                    "",
                    "| Scenario | Rate | RPS | p50 | p95 | p99 | Errors | Dropped | api_1 | api_2 | api_3 |",
                    "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
                ]
            )
        lines.append(
            "| {scenario} | {rate_rps} | {throughput_rps} | {p50_ms} | {p95_ms} | {p99_ms} | {error_rate} | {dropped} | {api_1_pct}% | {api_2_pct}% | {api_3_pct}% |".format(
                **row
            )
        )
    lines.append("")
    SUMMARY_MD.write_text("\n".join(lines))


def main():
    rows = summarize(group_runs(read_runs()))
    write_csv(rows)
    write_markdown(rows)
    print(f"wrote {SUMMARY}")
    print(f"wrote {SUMMARY_MD}")


if __name__ == "__main__":
    main()
