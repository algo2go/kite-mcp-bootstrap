# Bootstrap Decomposition — Empirical Mapping

**Date**: 2026-05-16 IST
**as-of**: 2026-05-16
**Master HEAD audited**: `8931b33` (Sprint 5 Pilot F-Y commit 2; B-series + Sprint 1-3 + Sprint 2-a + Pilot F all landed)
**Dispatch role**: empirical inventory + dep graph + decomposition shape options. Bootstrap-internal scope only — recommendation deferred to Audit's synthesis doc.

**Charter**: produce the empirical baseline that the strategic synthesis needs. NO recommendation on which shape to pick; NO source mutations.

---

## §INPUTS — Probe Methodology

Every load-bearing fact below is grounded in a probe. Listed source + probe + date per claim. All probes run at HEAD `8931b33`, 2026-05-16 IST.

| Claim | Source | Probe |
|---|---|---|
| 28 algo2go modules external | `go.work` workspace inspection | `cat go.work` |
| kc/ is NOT its own module | filesystem | `ls kc/go.mod` → no such file |
| testutil IS its own module | filesystem | `ls testutil/go.mod` → exists |
| app/metrics NOT its own module | filesystem | `ls app/metrics/go.mod` → no such file |
| File counts per mcp/subdir | filesystem | `ls mcp/$d/*.go ; wc -l mcp/$d/*.go` |
| Cross-subdir imports | git-tracked source | `grep -h "mcp/<dir>" mcp/<other>/*.go` (empirical leaf-decomp check) |
| 0 cross-subdir type leaks | full-tree grep | `for type in ...; do grep -rEn $type --include='*.go' . | grep -v "mcp/$d/"` |
| Per-subdir algo2go deps | grep imports | `grep -h '"github.com/algo2go' mcp/$d/*.go | sort -u` |
| kc.X usage per subdir | grep | `grep -ohE '\bkc\.[A-Z][a-zA-Z0-9]+' mcp/$d/*.go | sort -u` |
| Registration via `init()` | empirical scan | `grep -rEn 'plugin.RegisterInternalTool' mcp/` |
| Production tool count = 111 | prior verification | per `.research/research/sprint-5-pattern-d2-prep-2026-05-11.md` (Sprint 5 PREP, 2026-05-11) |

---

## §1 — File Counts + LOC per `mcp/<subdir>`

Non-test files (source) vs test files. All counts at HEAD `8931b33`.

| Subdir | Non-test files | Non-test LOC | Test files | Test LOC | Exported types | Handler impls | `init()` registrations |
|---|---|---|---|---|---|---|---|
| `mcp/admin/` | 8 | 1,791 | 0 | 0 | 22 | 18 | 18 |
| `mcp/alerts/` | 5 | 1,508 | 3 | 338 | 8 | 8 | 8 |
| `mcp/analytics/` | 6 | 4,195 | 10 | 2,190 | 10 | 8 | 8 |
| `mcp/misc/` | 4 | 918 | 0 | 0 | 10 | 9 | 9 |
| `mcp/paper/` | 5 | 1,326 | 0 | 0 | 12 | 8 | 8 |
| `mcp/portfolio/` | 9 | 2,380 | 2 | 394 | 21 | 20 | 20 |
| `mcp/trade/` | 9 | 3,710 | 0 | 0 | 30 | 28 | 28 |
| **subdir totals** | **46** | **15,828** | **15** | **2,922** | **113** | **99** | **99** |
| `mcp/` root (orchestrator + 12 root tools + shared) | 35 | 5,691 | ~20 | 32,653 | n/a | 12 | 29 |
| `mcp/common/` (leaf kernel) | 18 | 2,432 | 0 | 0 | n/a | 0 | 0 |
| `mcp/middleware/` (leaf) | 6 | 2,199 | 0 | 0 | n/a | 0 | 0 |
| `mcp/plugin/` (leaf) | 14 | 5,758 | 0 | 0 | n/a | 0 | 0 |
| **mcp/ tree total** | **119** | **31,908** non-test + ~32k test = 64,561 total LOC | | | | **111** | **128** |

**Notes**:
- 99 `RegisterInternalTool` calls in subdirs + 29 in root = 128 total registrations, but only 111 actually surface at runtime (rest are gated/excluded per `mcp/mcp.go:filterToolsWithGating`).
- The 32k+ LOC of tests at `mcp/` root is concentrated in ~12 big files (`tool_handlers_test.go` 1795 LOC, `tools_validation_test.go` 1476, etc.). These test the full tool surface from one place; per-subdir migration may need test-file follow-on.

---

## §2 — Cross-Import Graph

### §2.1 mcp/<subdir> → mcp/<other> imports

**0 cross-subdir imports** between any of the 7 tool subdirs. Each subdir is a perfect leaf:

```
mcp/admin     imports → {mcp/common, mcp/plugin}
mcp/alerts    imports → {mcp/common, mcp/plugin}
mcp/analytics imports → {mcp/common, mcp/plugin}
mcp/misc      imports → {mcp/common, mcp/plugin}
mcp/paper     imports → {mcp/common, mcp/plugin}
mcp/portfolio imports → {mcp/common, mcp/plugin}
mcp/trade     imports → {mcp/common, mcp/plugin}
```

### §2.2 mcp/<subdir> → mcp/ (root) imports

**0 subdir-to-root imports.** No tool subdir reaches back into the `mcp/` package root.

### §2.3 mcp/common imports

```
mcp/common imports →
  - stdlib (context, sync, fmt, ...)
  - external SDK: mark3labs/mcp-go
  - algo2go/kite-mcp-alerts
  - algo2go/kite-mcp-broker
  - algo2go/kite-mcp-cqrs
  - algo2go/kite-mcp-oauth
  - algo2go/kite-mcp-riskguard
  - algo2go/kite-mcp-usecases
  - algo2go/kite-mcp-users
  - bootstrap/kc            ← bootstrap-internal
  - bootstrap/kc/ports      ← bootstrap-internal
```

### §2.4 mcp/plugin imports

```
mcp/plugin imports →
  - stdlib
  - algo2go/kite-mcp-decorators
  - algo2go/kite-mcp-domain
  - bootstrap/mcp/common    ← intra-package (mcp/common still in bootstrap)
  - bootstrap/kc            ← bootstrap-internal
```

### §2.5 kc/ → algo2go imports + reverse coupling

kc/ imports 21 algo2go modules (alerts, audit, billing, broker, broker/mock, broker/zerodha, clockport, cqrs, domain, eventsourcing, instruments, money, papertrading, registry, riskguard, templates, ticker, usecases, users, watchlist).

**kc → bootstrap reverse coupling** (the lone blocker for kc-as-its-own-module):
- `kite-mcp-bootstrap/app/metrics` (4 production files: `kc/config.go`, `kc/manager_struct.go`, `kc/options.go`, `kc/scheduling_service.go`) — needs `*metrics.Manager` type
- `kite-mcp-bootstrap/testutil` (3 test files only) — needs `testutil.NewFakeClock`. `testutil` already has its own `go.mod`, so this is fine when kc/ becomes its own module.

**Dep graph rollup** (Mermaid-ish ASCII):

```
                 external algo2go modules (28 leaves)
                              ↑
                              ↑   (imports)
                              ↑
      ┌─────────── bootstrap/kc ─────────── bootstrap/app/metrics
      ↑                          ↑                     ↑
      ↑                          ↑                     │
      ↑                          ↑                     │
bootstrap/mcp/common ←────────── bootstrap/mcp/plugin  │
      ↑                          ↑                     │
      ↑          ┌───────────────┴────────┐            │
      ↑          ↑           ↑            ↑            │
   subdir(7) ←──┘           subdir(7)    subdir(7)    │
   mcp/admin/ ...             ↑                       │
                              ↑                       ↑
                       (no cross-subdir imports)      ↑
                                                      │
                       bootstrap/app/* ───────────────┘
                              ↑
                          bootstrap (root main)
```

---

## §3 — Shared Helpers Between Subdirs

### §3.1 What's in mcp/common (the shared kernel)

`mcp/common/` 2,432 non-test LOC across 18 files. Surface used by every subdir:

| File | LOC | What |
|---|---|---|
| `tool.go` | 96 (incl Tool2 add) | `Tool` + `Tool2` interface definitions |
| `handler_deps.go` | ~330 | `ToolHandler` + `ToolHandlerDeps` + 5-context builder composition + `NewToolHandlerFromDeps` |
| `handler_methods.go` | ~570 | `WithSession`, `WithViewerBlock`, `WithTokenRefresh`, `CallWithNilKiteGuard`, `SimpleToolHandler` |
| `handler_tracking.go` | ~50 | `TrackToolCall`, `TrackToolError`, `trackToolError` |
| `handler_response.go` | ~100 | `MarshalResponse`, `SanitizeData` |
| `argparser.go` | ~120 | `NewArgParser`, `ArgParser.String/Int/Float/Bool` |
| `validate.go` | ~30 | `ValidateRequired` |
| `cache.go`, `cache_lru.go` | ~250 | TTL cache + LRU |
| `elicit.go` | ~150 | Elicitation primitives |
| `integrity.go` | ~80 | Tool-description manifest |
| `argsanitize.go`, `errors.go`, `formats.go`, `pagination.go`, `retry.go`, `responsetypes.go` | ~250 each | Various |
| `session_deps.go`, `alert_deps.go`, `order_deps.go`, `admin_deps.go`, `read_deps.go` | ~50 each | The 5-context Deps builders (Investment K pattern) |

**Churn-prone vs stable**:
- **Stable**: `tool.go`, `argparser.go`, `validate.go`, `cache.go`, `elicit.go`, `integrity.go`. These are the leaf-level utilities. No subdir-specific churn.
- **Churn-prone**: `handler_deps.go`, the 5 `*_deps.go` builders. Every time a new Provider port is added, a builder gets a new field. But churn is ADDITIVE — existing fields stay stable. Cross-subdir agents adding new ports would still need to coordinate at this file.

### §3.2 What's in mcp/plugin

`mcp/plugin/` 5,758 non-test LOC. Holds the `Registry` struct (26 fields, 39 methods, 693 LOC per the original god-object inventory) + plugin discovery + 7 widget plugins.

**Subdirs depend on `plugin.RegisterInternalTool(t Tool)`** at `init()` time. That's the registry-coupling surface. Other plugin functions are app/-level wiring, not consumed by subdirs.

### §3.3 mcp/aliases.go (passthrough shims)

`mcp/aliases.go` 251 LOC — pure passthrough aliases:
- Type aliases: `Tool = common.Tool`, `ToolHandler = common.ToolHandler`, `ToolHandlerDeps = common.ToolHandlerDeps`, `TradingContext = paper.TradingContext`, etc.
- Function aliases: `NewToolHandler`, `NewToolHandlerFromDeps`, `NewArgParser`, `ValidateRequired`, etc.

This file is only needed by the ROOT-mcp tools (the 12 Pilot F + 17 others) — subdir tools import `common` directly. If sub-gits are extracted, `mcp/aliases.go` becomes the orchestrator's compat shim only.

---

## §4 — Registry Callsite Analysis

### §4.1 How tools currently register

Each tool file has an `init()` function:

```go
// e.g. mcp/admin/admin_anomaly_tool.go:327
func init() { plugin.RegisterInternalTool(&AdminListAnomalyFlagsTool{}) }
```

`plugin.RegisterInternalTool` appends the tool to a package-level slice in `mcp/plugin`. At server startup, `RegisterToolsForRegistry` (at `mcp/mcp.go:205`) iterates that slice and adds each tool to the live MCP server.

**128 `init()` registrations across the tree** (99 in subdirs + 29 at root). Production filters this to 111 via gating.

### §4.2 If trade/ becomes its own git, how does it register?

**Option A — Side-effect import** (cleanest):

```go
// in bootstrap main / app/wire.go
import (
    _ "github.com/algo2go/kite-mcp-trade"
    _ "github.com/algo2go/kite-mcp-portfolio"
    // ...
)
```

The sub-git's `init()` functions fire at import time, registering into `mcp/plugin.internalToolRegistry`. Provided `mcp/plugin` itself remains importable (either stays in bootstrap or also extracts), the registration model is unchanged.

**Option B — Explicit register call**:

```go
// each sub-git exports a Register func instead of init()
import tradetools "github.com/algo2go/kite-mcp-trade"

func wireUpTools() {
    tradetools.RegisterAll(plugin.DefaultRegistry)
    // ...
}
```

More explicit; less magic. But requires changing every existing tool from `func init()` to membership in an exported `RegisterAll` function.

**Option C — Plugin discovery via filesystem** (deferred — out of scope for normal decomp):

Already exists in `mcp/plugin/` for hashicorp-go-plugin subprocess plugins. Not a path for in-process tool migration.

**Tradeoffs**:
- A: zero source change per tool, side-effect import only. Risk: side-effect imports are easy to forget; missing one drops tools silently.
- B: explicit + linker-checked; harder to forget. Cost: ~28 `init()` → `RegisterAll` migrations per subdir.

### §4.3 mcp/plugin's role under decomposition

`mcp/plugin` is the registry mechanism. If sub-gits are extracted, `mcp/plugin` must EITHER:
1. Stay in bootstrap, with sub-gits importing `bootstrap/mcp/plugin` ← reverse coupling to bootstrap
2. Become its own module `algo2go/kite-mcp-plugin` ← clean leaf, every sub-git imports it

Option 2 is cleaner but requires `mcp/plugin` to fully shed `bootstrap/kc` (the `*kc.Manager` is in the legacy Tool interface). Once Tool2 supersedes Tool (post-coordinator PR Z), `mcp/plugin` could drop `kc/` and become a pure-stdlib + decorators + domain module.

---

## §5 — Six Residual manager.X() Refs from Pilot F PREP

Exact file:line for each (re-verified at HEAD `8931b33`):

| # | Location | Code | What needed | Provider port covers? |
|---|---|---|---|---|
| 1 | `mcp/admin/admin_baseline_tool.go:110` | `auditStore := manager.AuditStoreConcrete()` | Concrete `*audit.Store` for `UserOrderStats()` / `StatsCacheHitRate()` methods that are NOT on `AuditStoreInterface` | **NO** — needs either (a) `AuditStoreConcreteProvider` new port returning `*audit.Store`, OR (b) widen `AuditStoreInterface` to include the 2 forensics methods |
| 2 | `mcp/admin/admin_cache_info_tool.go:121` | same as #1 | same | same |
| 3 | `mcp/misc/session_admin_tools.go:93` | `reg := manager.SessionManager` (field access — was `SessionManager()` pre-B4) | `*SessionRegistry` for `ListActiveSessions()` / `TerminateByEmail()` methods | **NO** — needs `SessionRegistryProvider` new port returning `*SessionRegistry` (or `ListActiveSessions()` + `TerminateByEmail(email)` on a narrow port) |
| 4 | `mcp/misc/session_admin_tools.go:211` | same as #3 | same | same |
| 5 | `mcp/trade/options_greeks_tool.go:471` | `return a.manager.GetBrokerForEmail(email)` — inside `optionsAnalyzer` struct, `a.manager` is a STRUCT FIELD on the analyzer, not the Handler closure's `manager` | `BrokerResolverProvider` (`*kc.Manager` satisfies) | **YES** — already covered; `optionsAnalyzer` could store `kc.BrokerResolverProvider` instead of `*kc.Manager`. Same blast radius as a 1-line struct-field type change. |
| 6 | `mcp/trade/pretrade_tool.go:159` | `handler.Manager().Logger.Error(...)` | Logger from manager | **YES** — `handler.LoggerPort()` already exposed; swap to `handler.LoggerPort().Error(ctx, ...)`. One-line change. |

**Summary**: 2 refs (Provider already exists) are trivial cleanups. 4 refs need new Provider ports (1 audit-store-concrete, 1 session-registry — both are 1-method narrow ports). All 6 are addressable; none structurally block sub-git extraction once the new ports land.

---

## §6 — Per-Subdir Candidate Sub-Git Outlines

For each tool subdir, what would extracting it to its own `algo2go/kite-mcp-<name>` git look like? File-move counts, dep-pulls, effort estimate.

### §6.1 mcp/admin → algo2go/kite-mcp-admin-tools

| Aspect | Value |
|---|---|
| Files to move | 8 source + 0 test = 8 (1,791 LOC) |
| Tools (Handler impls) | 18 |
| algo2go modules needed | kite-mcp-audit, kite-mcp-billing, kite-mcp-cqrs, kite-mcp-domain, kite-mcp-riskguard, kite-mcp-usecases (6 modules) |
| bootstrap pulls | `bootstrap/kc` (or `kc/` as its own module first), `bootstrap/mcp/common`, `bootstrap/mcp/plugin` |
| Type-identity risk | LOW (22 exported types, 0 cross-subdir consumed per §2.5) |
| Residual blockers | `manager.AuditStoreConcrete()` × 2 refs (§5 #1-2) — need new Provider port BEFORE extract OR extract with manager-Handler still required |
| Test files to move | 0 (mcp/admin has no test files at subdir-level; tests live at mcp/ root) |
| Estimated effort | ~3-4h: create go.mod, move 8 files, update 6 algo2go deps in new go.mod, fix imports (mcp/admin → admin/), CI |

### §6.2 mcp/alerts → algo2go/kite-mcp-alerts-tools

| Aspect | Value |
|---|---|
| Files to move | 5 source + 3 test = 8 (1,508 + 338 LOC) |
| Tools | 8 |
| algo2go modules needed | kite-mcp-broker, kite-mcp-cqrs, kite-mcp-instruments, kite-mcp-oauth, kite-mcp-ticker (5 modules) |
| Type-identity risk | LOW (8 exported types, 0 cross-subdir consumed) |
| Residual blockers | None — fully typed-deps already |
| Test files | 3 in-subdir test files (good — tests move with code) |
| Estimated effort | ~2-3h |

### §6.3 mcp/analytics → algo2go/kite-mcp-analytics-tools

| Aspect | Value |
|---|---|
| Files to move | 6 source + 10 test = 16 (4,195 + 2,190 LOC) |
| Tools | 8 |
| algo2go modules needed | kite-mcp-broker (+ broker/mock), kite-mcp-cqrs, kite-mcp-domain, kite-mcp-money, kite-mcp-usecases (5 + mock = 6) |
| Type-identity risk | LOW (10 exported types) |
| Residual blockers | None |
| Test files | 10 in-subdir tests (heaviest test coverage — moves with code) |
| Estimated effort | ~4-5h (largest LOC volume) |

### §6.4 mcp/misc → algo2go/kite-mcp-misc-tools

| Aspect | Value |
|---|---|
| Files to move | 4 source + 0 test = 4 (918 LOC) |
| Tools | 9 |
| algo2go modules needed | kite-mcp-audit, kite-mcp-cqrs, kite-mcp-oauth, kite-mcp-riskguard, kite-mcp-ticker (5) |
| Type-identity risk | LOW |
| Residual blockers | `manager.SessionManager` × 2 refs (§5 #3-4) — need new Provider port |
| Test files | 0 |
| Estimated effort | ~2-3h |

### §6.5 mcp/paper → algo2go/kite-mcp-paper-tools

| Aspect | Value |
|---|---|
| Files to move | 5 source + 0 test = 5 (1,326 LOC) |
| Tools | 8 |
| algo2go modules needed | kite-mcp-audit, kite-mcp-cqrs, kite-mcp-domain, kite-mcp-oauth, kite-mcp-scheduler, kite-mcp-usecases (6) |
| Type-identity risk | **MEDIUM** — `paper.TradingContext` is type-aliased at `mcp/aliases.go:60` (`TradingContext = paper.TradingContext`). Used by 2 root-mcp test files. Extracting paper/ requires either (a) keeping the alias for the orchestrator's tests, or (b) migrating the 2 test files to import `algo2go/kite-mcp-paper-tools.TradingContext` directly. |
| Residual blockers | None in the handler bodies |
| Test files | 0 in subdir |
| Estimated effort | ~3h (includes the TradingContext alias migration) |

### §6.6 mcp/portfolio → algo2go/kite-mcp-portfolio-tools

| Aspect | Value |
|---|---|
| Files to move | 9 source + 2 test = 11 (2,380 + 394 LOC) |
| Tools | 20 |
| algo2go modules needed | kite-mcp-alerts, kite-mcp-broker, kite-mcp-cqrs, kite-mcp-money, kite-mcp-oauth, kite-mcp-sectors, kite-mcp-usecases (7) |
| Type-identity risk | LOW |
| Residual blockers | None |
| Test files | 2 in-subdir (pure_analytics_test.go covers backtest, indicators, sector mapping — moves with code) |
| Estimated effort | ~3-4h |

### §6.7 mcp/trade → algo2go/kite-mcp-trade-tools

| Aspect | Value |
|---|---|
| Files to move | 9 source + 0 test = 9 (3,710 LOC) |
| Tools | 28 |
| algo2go modules needed | kite-mcp-alerts, kite-mcp-broker, kite-mcp-cqrs, kite-mcp-domain, kite-mcp-instruments, kite-mcp-oauth, kite-mcp-ticker, kite-mcp-usecases (8 — broadest dep surface) |
| Type-identity risk | LOW (30 exported types, none cross-consumed) |
| Residual blockers | `options_greeks_tool.go:471` struct-field manager (§5 #5) — 1-line fix; `pretrade_tool.go:159` Logger access (§5 #6) — 1-line fix |
| Test files | 0 in subdir |
| Estimated effort | ~4-5h (largest tool count + dep surface) |

### §6.8 Aggregate (all 7 subdirs)

| Metric | Value |
|---|---|
| Total LOC moved | ~16,000 source + ~3,000 in-subdir tests ≈ 19,000 LOC |
| Total tools moved | 99 of 128 (77%) |
| Total go.mod files created | 7 new |
| Total algo2go module deps (unique union) | ~12 |
| Total dispatch effort if serial | ~22-27h aggregate |
| Total dispatch effort if parallel (7 agents on 7 gits) | ~5h wall-clock |

---

## §7 — Type-Identity Risk per Candidate Sub-Git

The type-identity scratch test (per established Path A promotion methodology): can a type defined in subdir A be passed to a function expecting the same-named type from a different package without an explicit conversion?

**Empirical scan**: for each exported type in each subdir, count references from OUTSIDE that subdir.

| Subdir | Exported types | Cross-subdir consumed | Verdict |
|---|---|---|---|
| `mcp/admin` | 22 | 0 | **CLEAN** |
| `mcp/alerts` | 8 | 0 | **CLEAN** |
| `mcp/analytics` | 10 | 0 | **CLEAN** |
| `mcp/misc` | 10 | 0 | **CLEAN** |
| `mcp/paper` | 12 | **1** (`paper.TradingContext` aliased at `mcp/aliases.go:60`; consumed by 2 root-mcp test files) | **MEDIUM** — handle via type alias in `mcp/aliases.go` for orchestrator compat OR migrate the 2 root-mcp test files |
| `mcp/portfolio` | 21 | 0 | **CLEAN** |
| `mcp/trade` | 30 | 0 | **CLEAN** |

**6 of 7 subdirs are clean leaves.** Paper has one bridgeable alias.

---

## §8 — Halt Conditions for the Decomposition Itself

What would block sub-git extraction?

1. **kc/ not its own module** (HARD blocker). Every sub-git's `mcp/common` import transitively pulls `bootstrap/kc`. If bootstrap is still where kc/ lives, sub-gits cannot live outside bootstrap unless they rewrite all their `bootstrap/kc.X` imports to a future `algo2go/kite-mcp-kc.X` path. **Prerequisite: extract kc/ as `algo2go/kite-mcp-kc` first.**

2. **app/metrics inside kc/'s import closure** (SOFT blocker). 4 kc/ source files import `bootstrap/app/metrics`. Either (a) extract `app/metrics` to its own algo2go module (likely small — pure-stdlib body), OR (b) define the `metrics.Manager` surface as a kc/ ports interface and inject the concrete type at wire time.

3. **mcp/common still inside bootstrap** (HARD blocker). Every sub-git imports `bootstrap/mcp/common`. **Prerequisite: extract `mcp/common` as `algo2go/kite-mcp-tools-common`** (or similar) — this is the migration ENABLER.

4. **mcp/plugin's `*kc.Manager` coupling via legacy Tool interface** (MEDIUM blocker). The `common.Tool` interface signature `Handler(*kc.Manager) server.ToolHandlerFunc` means `mcp/plugin` (which stores `[]Tool`) currently transitively requires `kc`. Either (a) wait for the Sprint 5 coordinator PR Z to retire the legacy Tool interface, OR (b) ship sub-gits with the legacy bridge in place (each tool's `Handler(*kc.Manager)` stays + uses `mcp/common.NewToolHandler(manager)` to construct deps; works as long as `mcp/common` is extracted alongside).

5. **128 init()-time registrations** (LOW risk, MEDIUM coordination cost). Each sub-git's `init()` registers into `mcp/plugin.internalToolRegistry`. If `mcp/plugin` is shared (extracted or in bootstrap), this works via side-effect imports — but every sub-git needs to be imported (e.g., `_ "github.com/algo2go/kite-mcp-trade-tools"`) in bootstrap's main wiring, or registrations silently drop.

6. **Test files at mcp/ root** (MEDIUM blocker for tests). 32,653 LOC of tests at `mcp/` root test the full tool surface from one place. Extracting sub-gits doesn't break the tests (Go can still build them as long as the root file imports the sub-gits), but per-subdir test files (mcp/alerts/*_test.go, mcp/analytics/*_test.go, mcp/portfolio/*_test.go) move with their code — test coverage per sub-git ends up uneven.

7. **mcp/ root has 12 tools + 29 init() registrations** (LOW blocker). After 7 subdirs extract, the root still has `watchlist_tools.go` (6), `market_tools.go` (5), `tax_tools.go` (1) = 12 tools. Need to either (a) extract these as the 8th sub-git, or (b) keep them at root as the orchestrator's vestige.

8. **Cyclic dep risk** (NONE observed). The leaf decomposition is acyclic per §2. No subdir imports another; no subdir imports root.

---

## §9 — Quantitative Summary Table

| Quantity | Count | Source |
|---|---|---|
| algo2go modules already external | 28 | `go.work` |
| Bootstrap-internal packages remaining | ~13 (kc, kc/ops, kc/ports, app, app/metrics, app/providers, mcp + 7 subdirs, mcp/common, mcp/middleware, mcp/plugin, testutil, plugins) | filesystem walk |
| Total mcp/ LOC (incl tests) | 64,561 | `find mcp -name '*.go' \| xargs wc -l` |
| mcp/ source LOC (non-test) | 31,908 | `find mcp -name '*.go' -not -name '*_test.go' \| xargs wc -l` |
| Tool subdirs (candidate sub-gits) | 7 | filesystem |
| Tool registrations in subdirs | 99 (of 128 total) | `grep -rEc 'plugin.RegisterInternalTool' mcp/<subdir>` |
| Cross-subdir imports | 0 | empirical grep |
| Cross-subdir type-leaks | 1 (`paper.TradingContext`) | per-type scan |
| Residual `manager.X()` refs needing new Provider ports | 4 (audit-concrete × 2 + session-registry × 2) | re-grep at HEAD |
| Residual `manager.X()` refs covered by existing Providers | 2 | same |
| kc → bootstrap reverse imports | 4 files (app/metrics) + 3 test files (testutil) | grep |
| mcp/common LOC | 2,432 | wc |
| mcp/plugin LOC | 5,758 | wc |

---

## §10 — What This Doc Does NOT Do

- **No recommendation on which decomposition shape to pick.** Audit's parallel research doc synthesizes the recommendation.
- **No source mutations.** This is the empirical baseline only.
- **No effort budget commitment.** Effort estimates are dispatch-side per-sub-git; sequencing + parallelism decisions are out of scope here.
- **No assessment of consumer-side impact.** What changes for `kite-mcp-server` (the consumer) when bootstrap decomposes is Audit's scope.
- **No comparison with B-series / Sprint 5 Pilot F outcomes.** Those are documented separately; this doc treats the current HEAD `8931b33` as the empirical starting point for the next decomposition.

---

**End of empirical mapping.**
