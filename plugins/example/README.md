# plugins/example — Sample MCP tool plugin

A minimal reference plugin demonstrating how to write an external MCP tool
that registers itself at startup via the `init()` side-effect import pattern.

## What it does

Defines `ServerTimeTool` — a single read-only MCP tool named `server_time`
that returns the current server time, timezone, and Unix timestamp.

This is purely educational; the production server has richer time/scheduling
tools elsewhere. Use this as a copy-paste starting point for your own plugin.

## How it gets registered

```go
// plugins/example/plugin.go
func init() {
    kitemcp.RegisterPlugin(&ServerTimeTool{})
}
```

Plugins are loaded by side-effect import in the bootstrap main wiring:

```go
import _ "github.com/algo2go/kite-mcp-bootstrap/plugins/example"
```

When the binary starts, `init()` runs and the tool is added to the MCP
tool registry. No central manifest; no per-build switch. The presence of
the import is the registration.

## Tool interface (3-method contract)

```go
type Plugin interface {
    Tool() mcp.Tool                                          // metadata
    Handler(manager *kc.Manager) server.ToolHandlerFunc      // execution
}
```

Optional 3rd method:

```go
// optional — runs once at registration time for stateful plugins
func (p *ServerTimeTool) Init(manager *kc.Manager) error { ... }
```

## Files

- `plugin.go` — the `ServerTimeTool` implementation (40 LOC)
- `plugin_test.go` — happy-path test asserting the tool returns RFC3339 time

## Writing your own plugin

1. Create `plugins/<yourplugin>/plugin.go`
2. Define a struct implementing `Tool()` + `Handler(manager *kc.Manager)`
3. Register in `init()` via `kitemcp.RegisterPlugin(&YourTool{})`
4. Add `_ "github.com/algo2go/kite-mcp-bootstrap/plugins/<yourplugin>"` to
   bootstrap's main wiring (or use the dynamic plugin-discovery path if
   running outside the monorepo)
5. Write a `plugin_test.go` with at minimum a happy-path test
6. Re-deploy; tool appears in `/healthz` `total_available` count

## Sibling plugins for further reading

- `plugins/rolegate/` — RBAC role-gate plugin (audit-logged authorization
  check before tool execution)
- `plugins/telegramnotify/` — Telegram-side-effect plugin (sends
  notifications after specific tool calls)

Both follow the same `init()` registration + `Tool()` + `Handler()` shape;
study them for production-quality examples that use middleware + per-user
state.

## Future relocation

This example currently lives in `bootstrap/plugins/example/` rather than
in the central `algo2go/kite-mcp-tools-common` module. When Phase 3
sub-git extraction completes (see kite-mcp-server `.research/phase-3-dispatch-briefs-2026-05-16.md`),
the plugin infrastructure remains in `kite-mcp-tools-common/plugin/` —
this example may relocate there as `kite-mcp-tools-common/plugin/example/`
to align with the canonical plugin-infrastructure home. Until then,
bootstrap is the correct location.

## See also

- `bootstrap/mcp/plugin/` — the plugin registry + lifecycle internals
- `algo2go/kite-mcp-tools-common/plugin/` (post-Phase-2) — the externalized
  plugin infrastructure module
- `bootstrap/CLAUDE.md` — plugin authoring conventions for the kite-mcp
  ecosystem
