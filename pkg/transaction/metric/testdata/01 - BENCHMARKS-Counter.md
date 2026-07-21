# Metric Collector Ingress Benchmarks — After Atomic Counter Change

This snapshot was captured **after** replacing `github.com/rcrowley/go-metrics`'s
`metrics.Counter` (interface + `StandardCounter`, itself backed by
`sync/atomic`) with an internal `*counter` type (`counter.go`) backed
directly on `atomic.Int64`. Same benchmark suite, same environment class, same
methodology as `BENCHMARKS-Start.md` — compare tables directly.

## Environment

- Date: 2026-07-21
- `go version`: go1.26.0 darwin/arm64
- CPU: Apple M4 Pro (`cpu: Apple M4 Pro` as reported by `go test -bench`)
- Command: `go test ./pkg/transaction/metric/... -run '^$' -bench 'BenchmarkAdd' -benchmem -benchtime=<N>x -count=1`

## Results across a load range

### BenchmarkAddMetric

| N    | ns/op | B/op | allocs/op | Δns/op vs baseline |
| ---- | ----: | ---: | --------: | -----------------: |
| 10   |  4679 | 1571 |        35 |              -4.3% |
| 100  |  3633 | 1531 |        36 |              +2.4% |
| 1000 |  3842 | 1550 |        37 |              +8.0% |
| 5000 |  3550 | 1540 |        37 |              -0.1% |

Still flat across load, as expected for a plain counter increment. Differences
are within run-to-run noise (single `-count=1` sample) — no measurable win or
regression from the atomic-int swap on this path.

### BenchmarkAddMetricDetail

| N    |  ns/op |   B/op | allocs/op | Δns/op vs baseline |
| ---- | -----: | -----: | --------: | -----------------: |
| 10   | 195900 | 120064 |      2211 |              +7.9% |
| 100  | 205793 | 122615 |      2307 |              +1.1% |
| 1000 | 249474 | 156261 |      3212 |              +3.7% |
| 5000 | 357062 | 250575 |      5472 |              +0.7% |

Same non-flat growth-with-N shape as baseline — confirms the growth there is
driven by the histogram/cache-update path, not the counter, since the counter
change didn't flatten it.

### BenchmarkAddAPIMetricDetail

| N    |   ns/op |    B/op | allocs/op | ns/transaction | Δns/op vs baseline |
| ---- | ------: | ------: | --------: | -------------: | -----------------: |
| 10   | 4205233 | 2533538 |     48047 |         210262 |              +0.4% |
| 100  | 6050230 | 4047554 |     83278 |         302512 |              -0.3% |
| 1000 | 8677752 | 5645678 |    120329 |         433888 |              +3.6% |
| 5000 | 8769396 | 5778147 |    123156 |         438470 |              +5.1% |

Within noise of baseline; no regression introduced.

### BenchmarkAddCustomMetricDetail

| N    |  ns/op |   B/op | allocs/op | Δns/op vs baseline |
| ---- | -----: | -----: | --------: | -----------------: |
| 10   | 200500 | 119781 |      2190 |              +8.4% |
| 100  | 194418 | 119112 |      2187 |              +0.5% |
| 1000 | 223294 | 119118 |      2187 |             +15.8% |
| 5000 | 202037 | 119139 |      2187 |              +6.6% |

Still flat allocs/op (unchanged from baseline: 2187 at steady state), so the
allocation profile of the counter path itself is unaffected. The ns/op moves
are noise from a single `-count=1` sample, not a systematic regression — this
path never touches the histogram code that dominates `AddMetricDetail`.

### BenchmarkAddAPIMetric

| N    | ns/op |  B/op | allocs/op | Δns/op vs baseline |
| ---- | ----: | ----: | --------: | -----------------: |
| 10   | 16846 | 10707 |       126 |              -4.8% |
| 100  | 10944 |  7829 |        75 |              -1.1% |
| 1000 | 12440 |  7694 |        70 |              +8.0% |
| 5000 | 11548 |  7769 |        70 |              +3.1% |

Flat, within noise, consistent with baseline (no aggregation path touched by
the counter change).

## Conclusion

Swapping `metrics.Counter` for an internal `atomic.Int64`-backed `*counter`
produced **no measurable performance change** across any ingress benchmark.
This is expected: `StandardCounter` was already a thin `atomic.Int64` wrapper
under an interface, so removing the interface indirection and the
`rcrowley/go-metrics` dependency for counters is a code-quality/dependency
win, not a throughput one. The real optimization target remains what the
baseline doc already identified — the histogram/cache-update path exercised
by `AddMetricDetail` / `AddAPIMetricDetail`, whose per-op cost still grows
with cumulative call volume in this snapshot.

All deltas above are single-sample (`-count=1`) and not statistically
rigorous; treat the percentages as rough noise bands, not a diff of means. For
that, re-run with `-count=10` on both revisions and use
[`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) per
`BENCHMARKS-Start.md`'s "How to compare" section.
