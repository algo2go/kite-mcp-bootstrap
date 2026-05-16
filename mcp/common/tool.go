// Package common holds the leaf-level mcp/ types and helpers — the
// Tool interface, the ToolHandler factory + its dependency container,
// argument-parsing utilities, response-marshalling helpers, the TTL
// cache, the elicitation primitives, and the tool-description integrity
// manifest. Anchor 1 PR 1.1 (Option B per .research/anchor-1-pr-1-1-
// redesign.md commit 34e5a23): split out from package mcp so the
// future per-domain sub-packages (mcp/middleware, mcp/plugin, mcp/admin,
// mcp/trade, mcp/portfolio, mcp/analytics, mcp/alerts, mcp/paper,
// mcp/misc) can all import the same shared kernel without depending
// on the mcp/ root.
//
// CYCLE-BREAK FRAMING
//
// The pre-PR-1.1 package layout had mcp.Tool (interface) and
// mcp.Registry (struct that holds []Tool) bidirectionally coupled
// inside the same package. The audit's redesign at 34e5a23 chose
// Option B — relocate the Tool interface alone to mcp/common, keep
// Registry-using functions (GetAllTools, GetAllToolsForRegistry) in
// mcp/ root. mcp/common ends up with zero imports of any
// kite-mcp-server/mcp* sub-package, while mcp/ root and the planned
// mcp/plugin (PR 1.3) both import mcp/common for the Tool type.
//
// One callback site needed restructuring (per Phase 3 of the redesign):
// mcp/common.go's `buildWriteTools()` previously called GetAllTools()
// from mcp/. Parameterising it as `BuildWriteTools(tools []Tool)`
// breaks the directional dependency cleanly — the mcp/ root passes
// the resolved tool slice in at startup.
//
// LEAF INVARIANT
//
// mcp/common imports only:
//   - stdlib (context, sync, fmt, ...)
//   - external mark3labs/mcp-go (the upstream MCP SDK)
//   - kite-mcp-server/kc and its already-extracted leaf modules
//     (kc/alerts, kc/cqrs, kc/logger, kc/ports, kc/riskguard,
//      kc/usecases, kc/users)
//   - kite-mcp-server/oauth (already-extracted module)
//
// It does NOT import any kite-mcp-server/mcp* path. A grep that
// returns non-empty results from `cd mcp/common && grep -rE
// "github.com/algo2go/kite-mcp-bootstrap/mcp\b" *.go` is a leaf-stability
// regression. Future PRs (1.2, 1.4-1.10) will add similar invariants
// for the per-domain sub-packages they extract.
package common

import (
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/algo2go/kite-mcp-bootstrap/kc"
)

// Tool is the contract every internal + external MCP tool implements.
// Tool() returns the wire-level descriptor (name, description, JSON
// schema); Handler returns the runtime function that the MCP server
// invokes when a client calls the tool by name.
//
// This interface was relocated from mcp/mcp.go as part of Anchor 1
// PR 1.1 (Option B per .research/anchor-1-pr-1-1-redesign.md). The
// `mcp` parent package keeps a `type Tool = common.Tool` alias so
// existing tool implementations (60+ at HEAD) compile unchanged.
type Tool interface {
	Tool() gomcp.Tool
	Handler(*kc.Manager) server.ToolHandlerFunc
}

// Tool2 is the typed-deps successor to Tool, introduced for the Sprint 5
// signature-flip migration. Implementations expose a HandlerDeps method
// that takes the narrow ToolHandlerDeps surface (27 Provider ports) in
// place of the full *kc.Manager. Tools migrating to Tool2 typically also
// retain their Handler(*kc.Manager) method as a 3-line bridge so the
// repo-wide Tool interface stays satisfied during the transition window:
//
//   func (*FooTool) Handler(m *kc.Manager) server.ToolHandlerFunc {
//       return (&FooTool{}).HandlerDeps(NewToolHandler(m).Deps)
//   }
//
// The registry callsite in mcp/mcp.go uses a runtime type-switch: if a
// tool implements Tool2 it dispatches through HandlerDeps with a shared
// ToolHandlerDeps built once per server registration; otherwise it falls
// back to the legacy Handler(*kc.Manager) path. Once every Tool also
// implements Tool2, the coordinator PR retires the legacy Tool interface
// (rename Tool2 -> Tool, drop the bridge methods, drop the type-switch).
//
// Empirical context: the 6th halt of the decomposition arc (2026-05-11)
// found that the 111 tool BODIES were already 95%+ typed-port-routed
// (only 6 residual manager.X() direct refs across the whole repo). The
// signature flip is the closing architectural cleanup — the additive
// Tool2 pattern lets it ship per-subdir in parallel instead of as one
// atomic 111-handler change.
type Tool2 interface {
	Tool() gomcp.Tool
	HandlerDeps(*ToolHandlerDeps) server.ToolHandlerFunc
}
