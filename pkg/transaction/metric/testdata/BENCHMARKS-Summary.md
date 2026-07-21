# Metric Collector Ingress Benchmarks — Progress Summary

This rolls up the five benchmark snapshots in this directory into one view of
how the ingress paths in `pkg/transaction/metric` have changed across the
optimization work so far. Same suite, same environment class, same
methodology throughout — see `00 - BENCHMARKS-Start.md`'s "How to reproduce /
compare" for the exact commands.

| #   | Snapshot                                                | Change                                                                                                                                                                                                         |
| --- | ------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 00  | [Start](00%20-%20BENCHMARKS-Start.md)                   | Baseline: `go-metrics` `Counter` (atomic-backed `StandardCounter`) + `Histogram` (2048-sample `UniformSample` reservoir)                                                                                       |
| 01  | [Counter](01%20-%20BENCHMARKS-Counter.md)               | Replaced `go-metrics` `Counter` with an internal `atomic.Int64`-backed `*counter`                                                                                                                              |
| 02  | [APICounter](02%20-%20BENCHMARKS-APICounter.md)         | Replaced `go-metrics` `Histogram` with an internal `*apiCounter` (count/min/max/sum only, no sample reservoir); `AddAPIMetricDetail` now reports stats directly instead of synthesizing + replaying 20 samples |
| 03  | [Registry](03%20-%20BENCHMARKS-Registry.md)             | Merged `metricMap` into the metric `registry` (generation-scoped grouped-metric keys, ack-deferred cleanup)                                                                                                    |
| 04  | [ContextResolve](04%20-%20BENCHMARKS-ContextResolve.md) | Deferred access-request/managed-app cache resolution from per-transaction `Add` time to once-per-template publish time; registry key now groups by raw app ID/name instead of resolved subscription/app IDs    |

## Environment (all snapshots)

- Date: 2026-07-21
- `go version`: go1.26.0 darwin/arm64
- CPU: Apple M4 Pro
- Command: `go test ./pkg/transaction/metric/... -run '^$' -bench 'BenchmarkAdd' -benchmem -benchtime=<N>x -count=1`

## ns/op across snapshots (N=5000, steady state)

| Benchmark                      |   Start | Counter | APICounter | Registry | ContextResolve | Net change (Start → ContextResolve) |
| ------------------------------ | ------: | ------: | ---------: | -------: | -------------: | ----------------------------------: |
| BenchmarkAddMetric             |    3555 |    3550 |       3670 |     3692 |           3653 |                       +2.8% (noise) |
| BenchmarkAddMetricDetail       |  354639 |  357062 |     188684 |   188980 |          10206 |                          **-97.1%** |
| BenchmarkAddAPIMetricDetail    | 8343711 | 8769396 |     193208 |   196690 |          10150 |                         **-99.88%** |
| BenchmarkAddCustomMetricDetail |  189542 |  202037 |     191437 |   189894 |           9290 |                          **-95.1%** |
| BenchmarkAddAPIMetric          |   11204 |   11548 |      11103 |    10960 |          11617 |                       +3.7% (noise) |

`AddAPIMetricDetail`'s ns/op is dominated by the 20-sample-per-call cost; see
the per-transaction row below for the comparable unit.

**Caveat on the ContextResolve column:** the benchmark's timed loop only covers
the `Add*` calls; `Execute()` (which now does the access-request/managed-app
cache lookup, once per metric template) runs after the timer stops. So this
drop is real but it's specifically an *ingress-cost* win — the lookup work
didn't disappear, it moved off the per-transaction path and is now amortized
once per template per publish cycle instead. See `04 - BENCHMARKS-ContextResolve.md`
for the full explanation.

## ns/transaction across snapshots (N=5000) — AddAPIMetricDetail only

The one benchmark that reports multiple transactions per call (`Count=20`),
so this is the fairer unit for comparing against `AddMetricDetail`'s
per-call cost.

| Snapshot       | ns/transaction |  Δ vs Start |
| -------------- | -------------: | ----------: |
| Start          |         417186 |           — |
| Counter        |         438470 |       +5.1% |
| APICounter     |           9660 |  **-97.7%** |
| Registry       |           9835 |      -97.6% |
| ContextResolve |          507.5 | **-99.88%** |

## allocs/op across snapshots (N=5000, steady state)

| Benchmark                      |  Start | Counter | APICounter | Registry | ContextResolve | Net change |
| ------------------------------ | -----: | ------: | ---------: | -------: | -------------: | ---------: |
| BenchmarkAddMetric             |     37 |      37 |         37 |       37 |             37 |          0 |
| BenchmarkAddMetricDetail       |   5472 |    5472 |       2209 |     2217 |            127 | **-97.7%** |
| BenchmarkAddAPIMetricDetail    | 123319 |  123156 |       2208 |     2216 |            126 | **-99.9%** |
| BenchmarkAddCustomMetricDetail |   2187 |    2187 |       2187 |     2200 |            123 | **-94.4%** |
| BenchmarkAddAPIMetric          |     70 |      70 |         70 |       70 |             70 |          0 |

## Growth-with-N: the shape that mattered most

The baseline's key finding was that `AddMetricDetail` / `AddAPIMetricDetail`
got *more expensive per op* as cumulative call volume (N) grew, even though
every call in the benchmark hits the same aggregation key — a red flag for
work scaling with total state rather than staying O(1) per call. The
histogram-removal change (`APICounter`) fixed this; everything since has
stayed flat.

### BenchmarkAddMetricDetail — ns/op by N

|    N |  Start | Counter | APICounter | Registry | ContextResolve |
| ---: | -----: | ------: | ---------: | -------: | -------------: |
|   10 | 181629 |  195900 |     195971 |   204158 |          12517 |
|  100 | 203633 |  205793 |     212881 |   197131 |           9761 |
| 1000 | 240522 |  249474 |     217020 |   186510 |          11616 |
| 5000 | 354639 |  357062 |     188684 |   188980 |          10206 |

Start/Counter roughly double from N=10 to N=5000 (growing with cumulative
state). From APICounter onward, cost is flat regardless of N — ContextResolve
keeps that same flat shape, just at a roughly 20x lower level, since removing
the resolution overhead does not reintroduce any per-N growth.

## What each change actually bought

- **Counter** (01): No measurable throughput change on any path. Expected —
  `go-metrics`' `StandardCounter` was already a thin `atomic.Int64` wrapper,
  so this was a dependency-removal / code-quality win, not a speed win.
- **APICounter** (02): The big one at the time. Removed the growth-with-N
  problem entirely and cut `AddAPIMetricDetail`'s per-transaction cost by
  ~95-98% (time, bytes, and allocations) by replacing a 2048-sample reservoir
  + 20x synthetic-sample replay with a single direct count/min/max/avg update.
- **Registry** (03): Neutral-to-small-cost. Merging `metricMap` into the
  registry doesn't touch `AddMetric` or `AddAPIMetric` at all, and adds a
  small, consistent overhead (+6 to +13 allocs/op, +200-450 B/op) to the
  three paths that use grouped metrics, from the extra lock-guarded `metrics`
  map and the `fmt.Sprintf`-built, start-time-suffixed registry key. This is
  architectural cleanup (single source of truth for grouped-metric state,
  ack-deferred cleanup) rather than a further throughput win.
- **ContextResolve** (04): The biggest ingress-cost win yet, and for a different
  reason than the others — it isn't a data-structure swap, it's a change in
  *when* work happens. Access-request/managed-app cache resolution (subscription,
  app, product, asset resource, API service revision, product plan, marketplace,
  quota — all the fields that need `agent.GetCacheManager()` lookups) used to run
  on every single `Add*` call. It now runs at most once per metric template, at
  publish time, and never again once it succeeds or is finalized with `unknown`
  placeholders. `AddMetricDetail`/`AddAPIMetricDetail`/`AddCustomMetricDetail`
  ingress cost drops 93-99.9% across time, bytes, and allocations as a result.
  This also fixed a correctness issue: the registry/cache key used to be built
  from the *resolved* subscription/app IDs, so two different apps that both
  failed to resolve (e.g. because the local cache wasn't warm yet) would
  collide into the same `unknown.unknown.<apiID>.<status>` group. The key is
  now built from the raw app ID/name instead, so grouping no longer depends on
  cache state at all.

## Where things stand

- `AddMetric` and `AddAPIMetric`: unaffected by any change so far, as
  expected — neither touches the histogram/counter, grouped-metric registry,
  or access-request/managed-app resolution code.
- `AddCustomMetricDetail`: ingress cost down ~95% from baseline (ContextResolve),
  after a small allocation creep (+13 allocs/op) from the Registry change that
  is now moot next to the much larger ContextResolve win.
- `AddMetricDetail` / `AddAPIMetricDetail`: the original optimization target,
  and now further improved by a second, independent axis of work (moving
  cache resolution off the hot path rather than changing the data structure
  it operates on). Growth-with-N remains eliminated and per-op cost/allocations
  are down 93-99.9% from baseline overall.
- The next place to look, if further ingress-cost reduction is wanted, is
  whatever remains in the `Add*` skeleton-building path itself (UUID
  generation, struct allocation, counter map lookups) — the access-request/
  managed-app resolution that used to dominate is no longer part of it.

All numbers above are single-sample (`-count=1`) as captured in the
individual snapshot docs; for a statistically rigorous comparison, use
`-count=10` with
[`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) per
`00 - BENCHMARKS-Start.md`'s "How to compare" section.
