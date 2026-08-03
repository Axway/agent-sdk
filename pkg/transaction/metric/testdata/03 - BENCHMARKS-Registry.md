# Metric Collector Ingress Benchmarks — After Registry/metricMap Merge

This snapshot was captured **after** merging the collector's `metricMap`
(a hand-rolled `sub -> app -> api -> status` nested map) into the metric
`registry`: `groupedMetrics` now also carries the `centralMetric` templates
that used to live in `metricMap`, the registry key for a group is suffixed
with the generation's start time (`metricStartTime`), and cleanup of a
group's registry entry is deferred until its events are actually acked
(instead of the old always-resident `metricMap` entries). Same benchmark
suite, same environment class, same methodology as `BENCHMARKS-Start.md` —
compare tables directly.

Deltas below are vs. `BENCHMARKS-Start.md` (the original go-metrics
baseline), per that doc's own convention (see `BENCHMARKS-APICounter.md`,
which also diffs against `Start` rather than the immediately preceding
snapshot). Note that the current codebase already includes the
`BENCHMARKS-Counter.md` and `BENCHMARKS-APICounter.md` changes, so the large
`AddMetricDetail` / `AddAPIMetricDetail` wins shown below are carried over
from that earlier histogram-removal work, not from this change — the
registry/metricMap merge itself is visible only in the small allocation/byte
deltas layered on top (see per-section notes).

## Environment

- Date: 2026-07-21
- `go version`: go1.26.0 darwin/arm64
- CPU: Apple M4 Pro (`cpu: Apple M4 Pro` as reported by `go test -bench`)
- Command: `go test ./pkg/transaction/metric/... -run '^$' -bench 'BenchmarkAdd' -benchmem -benchtime=<N>x -count=1`

## Results across a load range

### BenchmarkAddMetric

Unaffected — this path never touches the registry's grouped-metric entries.

| N    | ns/op | B/op | allocs/op | Δns/op | ΔB/op | Δallocs/op |
| ---- | ----: | ---: | --------: | -----: | ----: | ---------: |
| 10   |  4946 | 1564 |        35 |  +1.2% |  0.0% |       0.0% |
| 100  |  3760 | 1531 |        36 |  +6.0% |  0.0% |       0.0% |
| 1000 |  3443 | 1534 |        37 |  -3.2% |  0.0% |       0.0% |
| 5000 |  3692 | 1542 |        37 |  +3.9% | +0.1% |       0.0% |

Flat, within noise — as expected, since `AddMetric` only touches the
`transaction.count`/`transaction.volume` counters, not the grouped-metric
registry entries this change touched.

### BenchmarkAddMetricDetail

| N    |  ns/op |   B/op | allocs/op | Δns/op |  ΔB/op | Δallocs/op |
| ---- | -----: | -----: | --------: | -----: | -----: | ---------: |
| 10   | 204158 | 118385 |      2213 | +12.4% |  -1.4% |      +0.1% |
| 100  | 197131 | 118918 |      2216 |  -3.2% |  -3.1% |      -3.9% |
| 1000 | 186510 | 118809 |      2216 | -22.5% | -23.9% |     -31.0% |
| 5000 | 188980 | 118872 |      2217 | -46.7% | -52.5% |     -59.5% |

The large drop from N=1000-5000 is the already-documented histogram-removal
win (`BENCHMARKS-APICounter.md`), not this change — cross-checking against
that doc's numbers (e.g. N=5000: 188684 ns/op, 118606 B/op, 2209 allocs/op)
shows this change adds a small, consistent **+7-8 allocs/op and +200-300
B/op** on top at every N, from the registry key now being built with
`fmt.Sprintf` (start-time suffix) and `groupedMetrics` now also owning a
lock-guarded `metrics` map (replacing the plain nested `metricMap`).

### BenchmarkAddAPIMetricDetail

| N    |  ns/op |   B/op | allocs/op | ns/transaction | Δns/op |  ΔB/op | Δallocs/op |
| ---- | -----: | -----: | --------: | -------------: | -----: | -----: | ---------: |
| 10   | 189646 | 118387 |      2214 |           9482 | -95.5% | -95.3% |     -95.4% |
| 100  | 198699 | 118964 |      2216 |           9935 | -96.7% | -97.1% |     -97.3% |
| 1000 | 188016 | 118820 |      2216 |           9401 | -97.8% | -97.9% |     -98.2% |
| 5000 | 196690 | 118912 |      2216 |           9835 | -97.6% | -97.9% |     -98.2% |

Same story as `AddMetricDetail`: the ~95-98% drop vs `Start` is the
histogram-removal win already documented in `BENCHMARKS-APICounter.md`.
Cross-checking against that doc's numbers (e.g. N=5000: 193208 ns/op, 118637
B/op, 2208 allocs/op) isolates this change's own cost to the same **+6-8
allocs/op, +200-300 B/op** seen in `AddMetricDetail`.

### BenchmarkAddCustomMetricDetail

| N    |  ns/op |   B/op | allocs/op | Δns/op | ΔB/op | Δallocs/op |
| ---- | -----: | -----: | --------: | -----: | ----: | ---------: |
| 10   | 188708 | 119188 |      2200 |  +2.1% | +0.3% |      +0.6% |
| 100  | 195757 | 119530 |      2200 |  +1.2% | +0.3% |      +0.6% |
| 1000 | 189193 | 119525 |      2200 |  -1.9% | +0.4% |      +0.6% |
| 5000 | 189894 | 119575 |      2200 |  +0.2% | +0.4% |      +0.6% |

Flat across load, same as `Start` and `APICounter` — this path never grew
with N. The consistent **+13 allocs/op (2187 → 2200)** and **+0.3-0.4% B/op**
at every load level is this change's own cost: the custom-unit metric
template now lives in `groupedMetrics.metrics` (a lock-guarded map lookup)
instead of a plain nested `metricMap` entry, plus the start-time-suffixed
registry key build.

### BenchmarkAddAPIMetric

Unaffected — this path builds and publishes events directly, without going
through the grouped-metric registry at all.

| N    | ns/op |  B/op | allocs/op | Δns/op | ΔB/op | Δallocs/op |
| ---- | ----: | ----: | --------: | -----: | ----: | ---------: |
| 10   | 18492 | 10937 |       126 |  +4.5% | +3.1% |       0.0% |
| 100  | 12102 |  7815 |        75 |  +9.3% |  0.0% |       0.0% |
| 1000 | 11572 |  7687 |        70 |  +0.5% | +0.1% |       0.0% |
| 5000 | 10960 |  7773 |        70 |  -2.2% | +0.1% |       0.0% |

Flat, within noise, consistent with `Start` and `APICounter` — this path
doesn't touch the registry's grouped-metric entries.

## Conclusion

Compared against the `Start` baseline, the large improvements on
`AddMetricDetail` and `AddAPIMetricDetail` (up to ~97% lower ns/op and
allocs/op) are entirely attributable to the already-committed histogram
removal documented in `BENCHMARKS-APICounter.md`; this change doesn't touch
that path. Layered on top of `APICounter`'s numbers, merging `metricMap`
into the registry (generation-scoped grouped-metric keys, ack-deferred
cleanup) adds a small, consistent overhead isolated to the three ingress
paths that use grouped metrics:

| Path                  | Δallocs/op vs APICounter | ΔB/op vs APICounter |
| --------------------- | -----------------------: | ------------------: |
| AddMetricDetail       |                 +7 to +8 |          +200-300 B |
| AddAPIMetricDetail    |                 +6 to +8 |          +200-300 B |
| AddCustomMetricDetail |                      +13 |          +400-450 B |

`AddMetric` and `AddAPIMetric`, which never touch the grouped-metric
registry, show no change beyond run-to-run noise — confirming the added
cost is isolated to the registry/metricMap-merge code path and doesn't leak
into unrelated ingress paths. No throughput regression was observed on any
path; the added allocations come from the extra `metrics` map now carried by
`groupedMetrics` and the `fmt.Sprintf`-built, start-time-suffixed registry
key.

All numbers above are single-sample (`-count=1`); for a statistically
rigorous comparison, use `-count=10` with
[`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) per
`BENCHMARKS-Start.md`'s "How to compare" section.
