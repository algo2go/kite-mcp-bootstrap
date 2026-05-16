# kite-mcp-bootstrap

> Composition root + DI wiring for [kite-mcp-server](https://github.com/Sundeepg98/kite-mcp-server).

## What lives here

This module is the **composition root** of the Kite MCP Server stack. It contains:

| Package | Purpose |
|---|---|
| `app/` | DI wiring, HTTP mux, lifecycle, graceful restart, healthz |
| `app/providers/` | Adapter providers (alert, audit, billing, etc.) |
| `app/metrics/` | Per-tool latency + error metrics |
| `kc/` | Kite client manager, sessions, credential store, OAuth callback handler |
| `kc/ops/` | Admin + user dashboards, ops handlers, log buffer, scanner, payoff viz |
| `kc/ports/` | Port interface declarations (alert, credential, instrument, order, session) |
| `mcp/` | MCP tool registrations + middleware (audit, riskguard, elicitation, paper trading) |
| `plugins/` | Plugin scaffolding (example, rolegate, telegramnotify) |
| `testutil/` | Test fakes + fixtures (in-memory stores, clock, kite-server fake) |

## What does NOT live here

The 28 algo2go domain modules (broker, money, alerts, billing, riskguard,
papertrading, telegram, ...) are external dependencies, each in their own repo.
See `go.mod` for the canonical list.

The deploy artifacts (`Dockerfile`, `fly.toml`, `server.json`, `smithery.yaml`,
`funding.json`, `litestream.yml`, `.mcp.json`, `cmd/` operational binaries) live
in [kite-mcp-server](https://github.com/Sundeepg98/kite-mcp-server), which
imports this module and exposes only a thin `main.go`.

## Why a separate module?

Per `.research/research/github-transfer-bootstrap-2026-05-11.md` (the design
audit), separating composition root from deploy gives:

- **Reusability**: the composition root can be embedded by alternate front-ends
  (CLI variant, integration-test harness, future fork) without dragging deploy
  configuration along.
- **Versioning**: bootstrap follows semantic-version cadence; deploy repo tags
  per release.
- **Bounded blast radius for deploy audits**: the deploy repo audit surface
  becomes ~12 files instead of 261 .go files.
- **Algo2go brand alignment**: all Go source under `algo2go/*`; the deploy repo
  retains its `kite-mcp-server` name for the install URL stability.

## Status

`v0.1.0` — initial relocation from `kite-mcp-server` in-tree.

## Build

```
go build ./...
go test ./...
```

## License

MIT — see `LICENSE`.
