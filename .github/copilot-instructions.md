# Copilot Instructions for `agent-sdk`

## Big picture architecture
- `agent-sdk` is a library-first foundation for building Amplify discovery/traceability/compliance agents, not a single runnable service. Start from `pkg/cmd/root.go` and `pkg/agent/agent.go`.
- Agent startup flow is: `cmd.NewRootCmd(...)` -> `PreRunE initialize` (viper/env/config watch) -> `run` -> `initConfig` -> `agent.InitializeWithAgentFeatures(...)` -> `commandHandler`.
- Configuration and runtime state are centralized through SDK interfaces (`config.CentralConfig`, `config.AgentFeaturesConfig`) and the global agent runtime (`pkg/agent/agent.go`).
- Resource/event synchronization with Amplify Central happens through API clients and optional gRPC stream/watch paths (`pkg/agent/stream`, `pkg/watchmanager`).

## Key data/control flows
- Config sources merge in this order: command flags + env vars + config file paths (`pathConfig`, `path.config`, `.`), with live reload via `viper.WatchConfig()` in `pkg/cmd/root.go`.
- Config reload triggers `onConfigChange` -> `initConfig` -> optional `agent.GetConfigChangeHandler()` callback.
- On first init, SDK starts health/status subsystems, cache sync (`agent.CacheInitSync()`), and version-check jobs (`startVersionCheckJobs` in `pkg/cmd/agentversionjob.go`).
- Job orchestration is global and coordinated through `pkg/jobs`; continuous job failures pause pooled continuous jobs until healthy again.

## Build/test/dev workflows
- Use root `Makefile` targets: `make dep`, `make test`, `make test-sonar`, `make apiserver-generate`, `make protoc`.
- `make test` runs `go vet` + race-enabled tests and writes `gocoverage.out`, `test-output.log`, `report.xml`.
- `make test-sonar` uses filtered package list (`GO_PKG_LIST`) that intentionally excludes generated API server clients/models.
- Proto generation is Docker-based (`rvolosatovs/protoc`) and targets files under `proto/*.proto`; avoid hand-editing generated `.pb.go` files.
- API server models/clients under `pkg/apic/apiserver` are generated (`pkg/apic/apiserver/README.md`), not manually maintained.

## Project-specific coding patterns
- Add new agent config via `properties.Properties` APIs (`pkg/cmd/properties/properties.go`) and `config:"..."` tags; validation is expected through `ValidateCfg()`.
- If config can be sourced from agent resources, implement `ApplyResources(*v1.ResourceInstance)` in agent config types (see discovery/traceability docs).
- Keep bootstrap wiring in `pkg/cmd/root.go`; avoid ad-hoc init logic scattered across packages.
- For traceability/compliance command wiring, alias prefix behavior matters (`properties.SetAliasKeyPrefix(...)`).
- Health behavior is part of startup contract: `healthCheckTicker` blocks early run until checks pass or timeout.

## Integration points
- Amplify Central/API server integration: `pkg/apic`, generated API server resources in `pkg/apic/apiserver`.
- Auth/token flow: `pkg/apic/auth`, platform token requester initialized in `agent.handleCentralConfig`.
- Watch subscriptions over gRPC: `pkg/watchmanager` with `WatchTopic` resources and sequence-based catch-up support.
- Traceability publishing transports and event models are documented under `docs/traceability` and `pkg/transaction`/`pkg/traceability`.

## Guardrails for AI edits
- Do not manually edit generated artifacts in `pkg/apic/apiserver/**` or `*.pb.go`; regenerate via Makefile targets.
- Prefer extending existing interfaces/hooks (`InitConfigHandler`, `CommandHandler`, config validators, resource callbacks) instead of introducing parallel patterns.
- When adding flags/config, wire both property registration and parsing/validation paths; mirror existing tests in `pkg/cmd/*_test.go`.
- Keep changes compatible with downstream agents that inject build metadata through `pkg/cmd/version.go` linker vars.
