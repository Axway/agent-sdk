# Metric Collector Ingress Benchmarks — After Deferred Access Request/App Resolution

This snapshot was captured **after** moving the access request/managed application
cache lookup (previously done inline in `createMetric` on every `Add*` call) out of
the ingress path entirely. `createMetric` now only builds a metric skeleton from
data that's always available (raw API/app details, status/unit name) and stashes
that context on the metric; `resolveMetricContext` performs the actual cache
lookup (and fills in `Subscription`, `App`, `Product`, `AssetResource`,
`APIServiceRevision`, `ProductPlan`, `Marketplace`, `Quota`) once per metric
template, the first time a publish cycle visits it, not once per transaction. The
lookup is one-shot: whether or not the app/access-request resolve, the metric is
finalized (with `unknown` placeholders if not) and never re-resolved on a later
cycle. This also changed the registry/cache key itself: `centralMetric.getKey()`
now groups by the raw app ID/name instead of the resolved subscription/app IDs,
so an unresolved cache lookup can no longer collide two different apps into the
same `unknown` group. Same benchmark suite, same environment class, same
methodology as `00 - BENCHMARKS-Start.md` — compare tables directly.

## Important caveat: what this benchmark does and doesn't measure

`b.ResetTimer()`/`b.StopTimer()` in `metricscollector_bench_test.go` bracket only
the `Add*` call loop; `mc.Execute()` (which runs `resolveMetricContext` and does
the actual cache lookup) runs **after** `b.StopTimer()`. So the large drops below
are not a claim that the access-request/managed-app lookup itself got cheaper —
it's the same lookup, doing the same work. What changed is *when* it's paid: it
used to run once per transaction (inside the timed `Add` loop); now it runs once
per metric template per publish cycle (outside the timed loop, amortized across
however many transactions landed in that template). These numbers isolate the
ingress (`Add`) cost only, which is exactly what dropped.

## Environment

- Date: 2026-07-21
- `go version`: go1.26.0 darwin/arm64
- CPU: Apple M4 Pro (`cpu: Apple M4 Pro` as reported by `go test -bench`)
- Command: `go test ./pkg/transaction/metric/... -run '^$' -bench 'BenchmarkAdd' -benchmem -benchtime=<N>x -count=1`

Deltas below are vs. `00 - BENCHMARKS-Start.md` (the original go-metrics
baseline), per that doc's own convention.

## Results across a load range

### BenchmarkAddMetric

Unaffected — this path never touches `createMetric`/the grouped-metric registry.

| N    | ns/op | B/op | allocs/op | Δns/op |
| ---- | ----: | ---: | --------: | -----: |
| 10   |  4600 | 1564 |        35 |  -5.9% |
| 100  |  3845 | 1532 |        36 |  +8.4% |
| 1000 |  3558 | 1534 |        37 |  +0.0% |
| 5000 |  3653 | 1539 |        37 |  +2.8% |

Flat, within noise, as expected.

### BenchmarkAddMetricDetail

| N    | ns/op |  B/op | allocs/op |  Δns/op |  ΔB/op | Δallocs/op |
| ---- | ----: | ----: | --------: | ------: | -----: | ---------: |
| 10   | 12517 |  5355 |       125 |  -93.1% | -95.5% |     -94.3% |
| 100  |  9761 |  5258 |       125 |  -95.2% | -95.7% |     -94.6% |
| 1000 | 11616 |  5274 |       126 |  -95.2% | -96.6% |     -96.1% |
| 5000 | 10206 |  5285 |       127 |  -97.1% | -97.9% |     -97.7% |

Removing the per-add access-request/managed-app cache resolution (and the
subscription/product/asset-resource reference objects it built on every call)
takes this path from ~180-355k ns/op down to ~10-12k ns/op. Also notably flat
across N — the same shape `APICounter`/`Registry` already established, just at
a much lower baseline now that resolution isn't happening at all in this path.

### BenchmarkAddAPIMetricDetail

| N    | ns/op |  B/op | allocs/op | ns/transaction |  Δns/op |  ΔB/op | Δallocs/op |
| ---- | ----: | ----: | --------: | --------------: | ------: | -----: | ---------: |
| 10   | 13154 |  5369 |       126 |             658 |  -99.7% | -99.8% |     -99.7% |
| 100  | 10322 |  5268 |       126 |             516 |  -99.8% | -99.9% |     -99.8% |
| 1000 | 10914 |  5298 |       126 |             546 |  -99.9% | -99.9% |     -99.9% |
| 5000 | 10150 |  5289 |       126 |             508 |  -99.9% | -99.9% |     -99.9% |

The largest relative win, for the same reason `AddMetricDetail` dropped — this
path already removed the histogram/reservoir cost in `APICounter`; what's left
is almost entirely the per-add cache resolution this change removes. Per
transaction cost drops from ~207k-419k ns (`Start`) to ~500-660 ns.

### BenchmarkAddCustomMetricDetail

| N    | ns/op | B/op | allocs/op | Δns/op |  ΔB/op | Δallocs/op |
| ---- | ----: | ---: | --------: | -----: | -----: | ---------: |
| 10   | 10683 | 7054 |       123 | -94.2% | -94.1% |     -94.4% |
| 100  |  8819 | 6955 |       123 | -95.4% | -94.2% |     -94.4% |
| 1000 |  9879 | 6972 |       123 | -94.9% | -94.1% |     -94.4% |
| 5000 |  9290 | 6969 |       123 | -95.1% | -94.1% |     -94.4% |

Same story — this path called `getAccessRequestAndManagedApp`/`getQuota` on
every add before; now it only builds the skeleton and increments a counter.

### BenchmarkAddAPIMetric

Unaffected — this path builds a fully-resolved `APIMetric` directly and never
calls `createMetric`/the deferred resolution.

| N    | ns/op |  B/op | allocs/op | Δns/op |
| ---- | ----: | ----: | --------: | -----: |
| 10   | 19921 | 10812 |       126 | +12.5% |
| 100  | 10546 |  8023 |        75 |  -4.7% |
| 1000 | 12330 |  7895 |        70 |  +7.0% |
| 5000 | 11617 |  7974 |        70 |  +3.7% |

Flat, within the same noise band already established in prior snapshots (N=10
is warmup/setup noise, per `00 - BENCHMARKS-Start.md`).

## Conclusion

Deferring access-request/managed-application resolution from per-transaction
`Add` time to once-per-template publish time removes the single most expensive
piece of work left in the ingress paths that use it (`AddMetricDetail`,
`AddAPIMetricDetail`, `AddCustomMetricDetail`): the cache-manager lookup and the
reference objects (`Subscription`, `App`, `Product`, `AssetResource`,
`APIServiceRevision`, `ProductPlan`, `Marketplace`, `Quota`) it used to build on
every single call. Ingress cost for those three paths drops 93-99.9% across the
board (time, bytes, and allocations) vs. the original baseline, on top of the
histogram/registry wins already captured in `APICounter`/`Registry`.

This isn't free — the same resolution work still happens, just once per metric
template per publish cycle instead of once per transaction, so the real-world
win scales with how many transactions land on the same sub/app/api/status
combination between publish cycles (the more aggregation, the bigger the win;
a workload with no repeat keys sees the cost simply shifted, not eliminated).
It also changes retry behavior: if the cache isn't warm yet at publish time, the
metric is finalized with `unknown` placeholders and reported that way rather
than being held back, since the lookup is one-shot per template.

All numbers above are single-sample (`-count=1`); for a statistically rigorous
comparison, use `-count=10` with
[`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) per
`00 - BENCHMARKS-Start.md`'s "How to compare" section.
