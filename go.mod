module github.com/algo2go/kite-mcp-bootstrap

go 1.25.0

// Skeleton at v0.1.0 — populated by Phase 2 of the bootstrap-relocation
// dispatch (2026-05-16). Until source moves in, this module has no
// `require` block beyond the implicit stdlib.
//
// Source will be mass-moved from kite-mcp-server (Sundeepg98/kite-mcp-server):
//   - kc/, kc/ops/, kc/ports/    (manager + sessions + ops + ports)
//   - app/, app/providers/, app/metrics/  (DI wiring + Fx providers)
//   - mcp/                        (MCP tool registrations + middleware)
//   - plugins/                    (plugin scaffolding sub-module)
//   - testutil/                   (test fakes + fixtures sub-module)
//
// 28 algo2go/kite-mcp-* domain modules will be required at their current
// versions (matching kite-mcp-server's go.mod at HEAD when Phase 2 runs).
//
// Per .research/research/github-transfer-bootstrap-2026-05-11.md §2.3
// (the bootstrap design audit at commit 13888e1).
