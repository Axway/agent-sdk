# Metric Collector Ingress Benchmarks — Baseline

This is the baseline performance snapshot for the `Collector` ingress methods in
`pkg/transaction/metric`, captured **before** any optimization work on the
package. Benchmarks live in `metricscollector_bench_test.go`. Each one drives
the ingress method through the real publish path (real `collector`, real
`agent`/cache setup, a mock traceability client that captures published
events) and asserts the published data matches what was added, so a benchmark
failure means a correctness regression, not just a speed one.

Re-run this suite after making changes and compare against the numbers below
(see "How to compare" at the bottom).

## Environment

- Date: 2026-07-21
- `go version`: go1.26.0 darwin/arm64
- CPU: Apple M4 Pro (`cpu: Apple M4 Pro` as reported by `go test -bench`)
- Command: `go test ./pkg/transaction/metric/... -run '^$' -bench 'BenchmarkAdd' -benchmem -benchtime=<N>x -count=1`

`-benchtime=<N>x` pins the exact number of ingress calls per run (rather than
letting Go auto-calibrate), which is what lets us look at behavior across a
load range instead of a single data point.

## Results across a load range

N is the number of times the ingress method is called in a single benchmark
run before the collector executes/publishes.

### BenchmarkAddMetric

Single counter/volume update per call (usage aggregation only, no metric event).

| N    | ns/op | B/op | allocs/op |
| ---- | ----: | ---: | --------: |
| 10   |  4888 | 1564 |        35 |
| 100  |  3548 | 1531 |        36 |
| 1000 |  3557 | 1534 |        37 |
| 5000 |  3555 | 1540 |        37 |

Flat across load — cost is independent of N, as expected for a plain counter
increment.

### BenchmarkAddMetricDetail

Per-transaction histogram update; all N calls aggregate into a single grouped
histogram/metric event.

| N    |  ns/op |   B/op | allocs/op |
| ---- | -----: | -----: | --------: |
| 10   | 181629 | 120064 |      2211 |
| 100  | 203633 | 122700 |      2307 |
| 1000 | 240522 | 156188 |      3211 |
| 5000 | 354639 | 250464 |      5472 |

**Not flat** — per-op cost grows with N (roughly 2x from N=10 to N=5000), and
B/op and allocs/op grow with it. Since every call in this benchmark hits the
same API/App/status key, this points at per-call work that scales with
*cumulative* state (e.g. cache/storage bookkeeping done on every add) rather
than genuinely constant per-op work. Worth profiling as an optimization
candidate.

### BenchmarkAddAPIMetricDetail

Each call reports `Count=20` synthetic transactions (a batch of response-code
samples) via `buildDurations` + 20x `AddMetricDetail`.

| N    |   ns/op |    B/op | allocs/op | ns/transaction* |
| ---- | ------: | ------: | --------: | --------------: |
| 10   | 4189592 | 2535073 |     48060 |          209480 |
| 100  | 6068882 | 4048819 |     83276 |          303444 |
| 1000 | 8375018 | 5639702 |    120118 |          418751 |
| 5000 | 8343711 | 5769213 |    123319 |          417186 |

\* ns/op divided by the 20 transactions reported per call, for comparison
against `BenchmarkAddMetricDetail`'s per-transaction cost.

Same growth pattern as `AddMetricDetail` (expected, since it calls
`AddMetricDetail` under the hood 20x per call), plateauing by N=1000-5000.
This is the most expensive ingress method per transaction reported, which
tracks with it doing the most work per call (synthetic sample generation +
20x histogram updates).

### BenchmarkAddCustomMetricDetail

Each call adds `Count=3` to a custom unit counter; all calls aggregate into
one published custom-unit event.

| N    |  ns/op |   B/op | allocs/op |
| ---- | -----: | -----: | --------: |
| 10   | 184896 | 118827 |      2187 |
| 100  | 193372 | 119131 |      2187 |
| 1000 | 192856 | 119064 |      2187 |
| 5000 | 189542 | 119116 |      2187 |

Flat across load — unlike `AddMetricDetail`, allocs/op does not grow with N,
suggesting the growth seen there is tied to the histogram/response-metrics
path specifically, not general per-call bookkeeping.

### BenchmarkAddAPIMetric

Single add-then-publish path: builds one fully-formed `APIMetric` per call
(unique subscription/app/api per iteration, no aggregation), appends directly
to the event batch, then does one `Publish()` for all N events.

| N    | ns/op |  B/op | allocs/op |
| ---- | ----: | ----: | --------: |
| 10   | 17704 | 10604 |       126 |
| 100  | 11068 |  7815 |        75 |
| 1000 | 11519 |  7681 |        70 |
| 5000 | 11204 |  7767 |        70 |

Flat across load (N=10 is warmup/setup noise). This is the cheapest ingress
path per event, consistent with doing no cache/access-request resolution and
no aggregation.

## Summary / where to look first

- `AddMetric` and `AddCustomMetricDetail`: flat cost, no obvious regression
  target.
- `AddMetricDetail` / `AddAPIMetricDetail`: per-op cost and allocations grow
  with cumulative call volume even though all calls hit the same aggregation
  key. This is the primary target for the upcoming optimization work — likely
  in the histogram/cache-update path (`createOrUpdateHistogram`,
  `updateMetricWithCachedMetric`, or `storage.updateMetric`).
- `AddAPIMetric`: cheapest per event, flat, no aggregation — low priority.

## How to reproduce / compare after changes

```sh
# full range used for this baseline
for n in 10 100 1000 5000; do
  go test ./pkg/transaction/metric/... -run '^$' -bench 'BenchmarkAdd' \
    -benchmem -benchtime=${n}x -count=1 | tee -a bench_after.txt
done
```

For a statistically rigorous comparison (recommended once real changes land),
capture `-count=10` at a fixed `-benchtime` before and after your change and
diff with [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```sh
go test ./pkg/transaction/metric/... -run '^$' -bench 'BenchmarkAdd' -benchmem -count=10 > before.txt
# ...make your change...
go test ./pkg/transaction/metric/... -run '^$' -bench 'BenchmarkAdd' -benchmem -count=10 > after.txt
go run golang.org/x/perf/cmd/benchstat@latest before.txt after.txt
```
