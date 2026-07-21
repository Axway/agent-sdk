# Metric Collector Ingress Benchmarks — After apiCounter (Histogram Removal)

This snapshot was captured **after** replacing `github.com/rcrowley/go-metrics`'s
`metrics.Histogram` (backed by a 2048-sample `UniformSample` reservoir) with an
internal `*apiCounter` type (`apicounter.go`) that tracks only `count`, `min`,
`max`, and a running sum — no per-sample retention. This also removed the
`buildDurations` synthetic-sample generator: `AddAPIMetricDetail` now feeds
count/min/max/avg directly into the counter in one call instead of
synthesizing N samples and replaying them one at a time through
`AddMetricDetail`. Same benchmark suite, same environment class, same
methodology as `BENCHMARKS-Start.md` — compare tables directly.

## Environment

- Date: 2026-07-21
- `go version`: go1.26.0 darwin/arm64
- CPU: Apple M4 Pro (`cpu: Apple M4 Pro` as reported by `go test -bench`)
- Command: `go test ./pkg/transaction/metric/... -run '^$' -bench 'BenchmarkAdd' -benchmem -benchtime=<N>x -count=1`

## Results across a load range

Deltas below are vs. `BENCHMARKS-Start.md` (the original go-metrics baseline),
since that's the meaningful before/after for this change — `BENCHMARKS-Counter.md`
already showed the atomic-counter swap didn't move these two paths.

### BenchmarkAddMetricDetail

Per-transaction counter update; all N calls aggregate into a single grouped
`apiCounter`/metric event.

| N    |  ns/op |   B/op | allocs/op | Δns/op |  ΔB/op | Δallocs/op |
| ---- | -----: | -----: | --------: | -----: | -----: | ---------: |
| 10   | 195971 | 118189 |      2206 |  +7.9% |  -1.6% |      -0.2% |
| 100  | 212881 | 118508 |      2207 |  +4.5% |  -3.4% |      -4.3% |
| 1000 | 217020 | 118590 |      2209 |  -9.8% | -24.1% |     -31.2% |
| 5000 | 188684 | 118606 |      2209 | -46.8% | -52.6% |     -59.6% |

**Flat now, where it used to grow.** Baseline cost roughly doubled from N=10
to N=5000 (181629 → 354639 ns/op) because every add serialized/replayed a
sample reservoir whose effective size grew with cumulative call volume. The
new counter has no reservoir, so cost stays flat (~190-220k ns/op) regardless
of how many transactions have already been aggregated into the same key — the
growth identified as the top optimization target in the baseline doc is gone.

### BenchmarkAddAPIMetricDetail

Each call reports `Count=20` transactions directly via `UpdateWithStats`
(count, min, max, avg in one shot — no more synthetic sample generation or
20x `AddMetricDetail` replay).

| N    |  ns/op |   B/op | allocs/op | ns/transaction | Δns/op |  ΔB/op | Δallocs/op |
| ---- | -----: | -----: | --------: | -------------: | -----: | -----: | ---------: |
| 10   | 190046 | 118202 |      2206 |           9502 | -95.5% | -95.3% |     -95.4% |
| 100  | 213039 | 118562 |      2207 |          10652 | -96.5% | -97.1% |     -97.4% |
| 1000 | 225846 | 118621 |      2208 |          11292 | -97.3% | -97.9% |     -98.2% |
| 5000 | 193208 | 118637 |      2208 |           9660 | -97.7% | -97.9% |     -98.2% |

This is the single biggest win from the change. Baseline cost for this path
(4.2M–8.3M ns/op) was dominated by generating 20 synthetic duration samples
per call and replaying them one at a time through the full `AddMetricDetail`
path (20x lock/unlock, 20x counter update, 20x cache-metric JSON
serialization of a growing sample array). Reporting count/min/max/avg
directly collapses that to one counter update and one cache write per call,
and per-transaction cost drops from ~209k-419k ns down to ~9.5k-11.3k ns
(roughly a 20-40x speedup per transaction). It's also now flat across load
instead of growing, for the same reservoir-removal reason as
`AddMetricDetail`.

### BenchmarkAddMetric / BenchmarkAddCustomMetricDetail / BenchmarkAddAPIMetric

Unaffected, as expected — none of these paths touch the counter/histogram
code.

| Benchmark                      | N    |  ns/op | Δns/op vs baseline |
| ------------------------------ | ---- | -----: | -----------------: |
| BenchmarkAddMetric             | 5000 |   3670 |              +3.2% |
| BenchmarkAddCustomMetricDetail | 5000 | 191437 |              +1.0% |
| BenchmarkAddAPIMetric          | 5000 |  11103 |              -0.9% |

All within run-to-run noise of the baseline and counter snapshots — no
regression on the paths this change didn't touch.

## Conclusion

Replacing the go-metrics `Histogram`/`UniformSample` reservoir with the
count/min/max/avg-only `apiCounter`, and replacing `buildDurations`'
synthetic-sample generation with a direct stats update, did what the baseline
doc flagged as the top optimization target: it eliminated the per-op cost
growth tied to cumulative call volume on `AddMetricDetail` and
`AddAPIMetricDetail`, and cut `AddAPIMetricDetail`'s per-transaction cost by
roughly 95-98% (allocations included) at every load level tested. Unlike the
counter-only change in `BENCHMARKS-Counter.md`, this is both a
dependency-removal win **and** a substantial throughput/memory win, since the
old histogram's fixed-size sample reservoir (and its JSON serialization on
every cache write) was the actual source of the scaling problem, not just
unnecessary interface indirection.

All numbers above are single-sample (`-count=1`); for a statistically
rigorous comparison, use `-count=10` with
[`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) per
`BENCHMARKS-Start.md`'s "How to compare" section.
