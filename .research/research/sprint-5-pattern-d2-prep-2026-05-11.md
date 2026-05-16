# Sprint 5 Pattern D.2 — Tool.Handler Signature Flip PREP

**Date**: 2026-05-11 IST
**as-of**: 2026-05-11
**Master HEAD audited**: `2b96ef6` (`refactor(manager-cqrs): split 788-LOC manager_commands_admin.go into 6 per-domain files (Sprint 3 Option-a)`)
**Dispatch role**: empirical survey + per-module migration brief template for the Tool.Handler signature flip (Pattern D.2 from Day-1 ops' blocker survey `f81c91b`). READ-ONLY pre-execution prep.

**Charter**: empirically count the actual Tool.Handler callsites, assess per-tool migration state, identify the actual remaining work, and design the per-module migration brief template for fan-out to parallel agents.

**Methodology**: compile-and-grep over the current repo (HEAD `2b96ef6`). Every numeric claim is grounded in a probe; counts cross-checked against production `total_available=111` from `/healthz`.

---

## §1 — Headline Empirical Finding

**The Tool.Handler migration to typed `*ToolHandlerDeps` is ALREADY 95%+ COMPLETE.**

The Day-1 ops blocker survey at `f81c91b` flagged 128 Tool.Handler callsites and estimated 120-250h for full migration. Empirical recount: **111 Tool.Handler implementations** (matches production tool count exactly), with **only 6 production `manager.X()` direct-coupling refs remaining** across the entire 111-handler corpus.

| Metric | Brief estimate | Empirical |
|---|---|---|
| Tool.Handler implementations | 128 | **111** |
| Production tool count (`/healthz total_available`) | — | 111 (matches) |
| `NewToolHandler(manager)` callsites (standard entry pattern) | — | 100+ |
| Direct `manager.X()` refs inside handler bodies | 128 | **6** (incl. 2 in admin "forensics-only" escape hatches + 1 documented "Provider-resolution" idiom) |
| `h.manager.X()` indirect refs in handler bodies | — | **0** (only in docstrings) |
| `handler.Manager().X()` indirect refs | — | **1** (`pretrade_tool.go` Logger access) |
| Variant Handler signatures (anything not `manager *kc.Manager`) | — | **0** |
| Tools per directory matching standard signature | — | 100% (99/99 in subdirs, 12/12 in root files) |

**Implication**: The substantive migration work has ALREADY happened, distributed across the Anchor 6 PR 6.4 / Wave D Package 7c / Phase 3a Batch 6 commits. What remains is a SIGNATURE FLIP from `Handler(manager *kc.Manager) server.ToolHandlerFunc` to `Handler(deps *common.ToolHandlerDeps) server.ToolHandlerFunc` (Pattern D.2) — purely mechanical at 111 callsites + the residual 5 manager.X() sites to either preserve via narrow-port or thread through Deps.

**Revised effort estimate**: **20-40h cumulative** (NOT 120-250h), distributable per algo2go module via 7-8 parallel agents. Wall-clock: ~4-8h.

---

## §2 — Per-Directory Tool Inventory + Migration State

Distribution of the 111 Tool.Handler implementations (empirical at HEAD `2b96ef6`):

| Directory | Handlers | `manager.X()` direct refs | LOC | algo2go modules used | Migration state |
|---|---|---|---|---|---|
| `mcp/trade/` | **28** | 1 (`options_greeks_tool.go:471` — `manager.GetBrokerForEmail` via Provider) | 3,710 | alerts, broker, cqrs, domain, instruments, oauth, ticker, usecases | ~99% typed-deps |
| `mcp/portfolio/` | **20** | 0 | 2,380 | alerts, broker, cqrs, money, oauth, sectors, usecases | 100% typed-deps |
| `mcp/admin/` | **18** | 2 (`admin_baseline_tool.go:110`, `admin_cache_info_tool.go:121` — both `AuditStoreConcrete()` forensics-only escape hatches) | 1,791 | audit, billing, cqrs, domain, riskguard, usecases | 100% typed-deps (refs are documented escape hatches) |
| `mcp/misc/` | **9** | 2 (`session_admin_tools.go:93,211` — `manager.SessionManager` field access for SessionRegistry directly) | 918 | audit, cqrs, oauth, riskguard, ticker | ~98% typed-deps |
| `mcp/alerts/` | **8** | 0 | 1,508 | broker, cqrs, instruments, oauth, ticker | 100% typed-deps |
| `mcp/analytics/` | **8** | 0 | 4,195 | broker, cqrs, domain, money, usecases | 100% typed-deps |
| `mcp/paper/` | **8** | 0 | 1,326 | audit, cqrs, domain, oauth, scheduler, usecases | 100% typed-deps |
| `mcp/watchlist_tools.go` (root) | 6 | 0 | 575 | — | 100% typed-deps |
| `mcp/market_tools.go` (root) | 5 | 0 | 409 | — | 100% typed-deps |
| `mcp/tax_tools.go` (root) | 1 | 0 | 327 | — | 100% typed-deps |
| **Total** | **111** | **6** | **17,139** | 8 distinct algo2go modules | **~99% typed-deps complete** |

**Three patterns observed in the 6 remaining manager.X() refs:**

1. **Provider-interface idiom (1 ref)**: `mcp/trade/options_greeks_tool.go:471` calls `a.manager.GetBrokerForEmail(email)` — `a` is a struct that holds a `*kc.Manager`, and `GetBrokerForEmail` is a `BrokerResolverProvider` method (kept per the B3 / Anchor 6 PR 6.4 preservation rule). Already correctly factored.

2. **Concrete-store escape hatches (2 refs)**: `mcp/admin/admin_baseline_tool.go:110` + `mcp/admin/admin_cache_info_tool.go:121` use `manager.AuditStoreConcrete()` — documented "forensics-only" path that needs the concrete type for methods NOT on `AuditStoreInterface` (e.g., `UserOrderStats`, `StatsCacheHitRate`). Per Sprint 1 Slice 3 audit-doc rationale, these are deliberate carve-outs.

3. **Field access on the post-B4 SessionManager (2 refs)**: `mcp/misc/session_admin_tools.go:93,211` use `manager.SessionManager` — Manager-field access for the SessionRegistry (which has methods like `ListActiveSessions` + `TerminateByEmail` that aren't on a Provider port). Could route via `h.Deps.Sessions.Registry()` if such a port existed; currently the directest path. Migration: would require adding a `SessionRegistryProvider` interface to `kc/manager_interfaces.go` and a corresponding field on `ToolHandlerDeps`.

4. **Logger-error access via Manager() (1 ref)**: `mcp/trade/pretrade_tool.go:159` uses `handler.Manager().Logger.Error(...)` — could swap to `handler.LoggerPort()` (already exposed) with ctx threading.

---

## §3 — The `ToolHandlerDeps` Surface (already in place)

The migration target — `common.ToolHandlerDeps` (at `mcp/common/handler_deps.go`) — already exposes **27 typed Provider ports**:

| Port | Type | Backing field on `*Manager` |
|---|---|---|
| `LoggerPort` | `logport.Logger` | `Logger` (port-wrapped) |
| `TokenStore` | `kc.TokenStoreInterface` | `tokenStore` (via TokenStore() method) |
| `UserStore` | `kc.UserStoreInterface` | `userStore` (via UserStore() method) |
| `Sessions` | `ports.SessionPort` | `SessionSvc` (port-wrapped) |
| `Credentials` | `ports.CredentialPort` | `CredentialSvc` (port-wrapped) |
| `Metrics` | `kc.MetricsRecorder` | `metrics` |
| `Config` | `kc.AppConfigProvider` | (Provider via DevMode/AppMode/ExternalURL methods) |
| `Tokens` | `kc.TokenStoreProvider` | (Provider on Manager) |
| `CredStore` | `kc.CredentialStoreProvider` | (Provider on Manager) |
| `Browser` | `kc.BrowserOpener` | (Provider on Manager) |
| `Alerts` | `ports.AlertPort` | `alertStore` (port-wrapped) |
| `Telegram` | `kc.TelegramStoreProvider` | (Provider on Manager) |
| `TelegramNotifier` | `ports.AlertPort` | `telegramNotifier` (port-wrapped) |
| `Watchlist` | `kc.WatchlistStoreProvider` | (Provider on Manager) |
| `Users` | `kc.UserStoreProvider` | (Provider on Manager) |
| `Registry` | `kc.RegistryStoreProvider` | (Provider on Manager) |
| `Audit` | `kc.AuditStoreProvider` | (Provider on Manager) |
| `Billing` | `kc.BillingStoreProvider` | (Provider on Manager) |
| `Ticker` | `kc.TickerServiceProvider` | (Provider on Manager) |
| `Paper` | `kc.PaperEngineProvider` | (Provider on Manager) |
| `Instruments` | `ports.InstrumentPort` | `Instruments` (port-wrapped) |
| `AlertDB` | `ports.AlertPort` | `alertDB` (port-wrapped) |
| `RiskGuard` | `kc.RiskGuardProvider` | (Provider on Manager) |
| `MCPServer` | `kc.MCPServerProvider` | (Provider on Manager — preserved per B3) |
| `BrokerResolver` | `kc.BrokerResolverProvider` | (Provider on Manager — Anchor 6 PR 6.4) |
| `TrailingStop` | `ports.AlertPort` | `trailingStopMgr` (port-wrapped) |
| `Events` | `kc.EventDispatcherProvider` | (Provider on Manager) |
| `PnL` | `ports.AlertPort` | (port-wrapped) |
| `CommandBusP` | `kc.CommandBusProvider` | (Provider on Manager — preserved per B5a) |
| `QueryBusP` | `kc.QueryBusProvider` | (Provider on Manager — preserved per B5b) |

**Composition path**: `NewToolHandler(manager)` composes Deps via 5 per-context builders (`newSessionDeps`, `newAlertDeps`, `newOrderDeps`, `newAdminDeps`, `newReadDeps`) — Investment K pattern. Adding a new field touches only the relevant context's builder; the constructor body is stable.

**Gap**: `SessionRegistryProvider` is the only port that would be needed but does not yet exist (for the 2 `manager.SessionManager` callsites in `misc/session_admin_tools.go`). Would be a 5-line addition.

---

## §4 — Pattern D.2 Migration Mechanics

The signature flip is from:

```go
// Current (every handler)
func (*FooTool) Handler(manager *kc.Manager) server.ToolHandlerFunc {
    handler := common.NewToolHandler(manager)
    return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
        // body uses handler.Deps.X or handler.X() — already typed-port-routed
    }
}
```

To:

```go
// Target (Pattern D.2)
func (*FooTool) Handler(deps *common.ToolHandlerDeps) server.ToolHandlerFunc {
    return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
        // body uses deps.X directly
    }
}
```

**Per-tool transformation steps** (mechanical):

1. Change signature: `Handler(manager *kc.Manager)` → `Handler(deps *common.ToolHandlerDeps)`.
2. Drop the `handler := common.NewToolHandler(manager)` line at the top of each Handler body.
3. Replace `handler.X` → `deps.X` (where `X` is a field on `ToolHandlerDeps`) or `handler.Y()` → use the appropriate `deps.Y` field.
4. Drop the `*kc.Manager` import if no longer used (likely needed for `kc.KiteSessionData` type references inside callback functions; verify per-tool).
5. For the 6 residual `manager.X()` refs, route through Deps OR through narrow Provider (per §2 patterns 1-3).

**Caller-side change**: every `Tool.Handler(manager)` invocation at the registry / `RegisterTools*` callsite must change to `Tool.Handler(deps)`. The single callsite is at `mcp/mcp.go` / `mcp/registry.go` (where tools are looped + their `.Handler()` is called). One mass-edit; should compile uniformly after the 111 method signatures change.

**Test-side change**: tool-handler tests currently call `tool.Handler(manager)(ctx, request)`. They must change to `tool.Handler(deps)(ctx, request)` after composing `deps` via `common.NewToolHandler(manager).Deps`. Test helpers in `testutil/kcfixture/` likely need an `OptDeps()` helper.

---

## §5 — Per-Algo2go-Module Ownership Map for Sprint 5 Fan-Out

The 28 algo2go modules + how the 111 tools partition by module. Modules with the most tool-coupling get their own agent dispatch; modules with no tool coupling are out-of-scope.

| Module | Tools touching it | Primary directory | Sprint 5 dispatch grouping |
|---|---|---|---|
| **cqrs** | 100+ (via QueryBus/CommandBus dispatch) | all dirs | **Cross-cutting** — handled at signature-flip layer, no per-module work |
| **broker** | ~40 | `mcp/trade`, `mcp/portfolio`, `mcp/alerts`, `mcp/analytics` | **Agent A** |
| **usecases** | ~50 | `mcp/admin`, `mcp/analytics`, `mcp/paper`, `mcp/portfolio`, `mcp/trade` | **Cross-cutting (CQRS)** |
| **alerts** | 8 | `mcp/alerts`, `mcp/trade` (alerts.Store types) | **Agent B** |
| **audit** | 18 + admin/misc | `mcp/admin`, `mcp/misc`, `mcp/paper` | **Agent C** |
| **billing** | 18 admin | `mcp/admin` | **Agent C** (with audit) |
| **domain** | 80+ event refs | all dirs | **Cross-cutting (event payloads)** |
| **instruments** | ~20 | `mcp/alerts`, `mcp/analytics`, `mcp/trade` | **Agent A** (with broker) |
| **money** | 28 | `mcp/analytics`, `mcp/portfolio` | **Agent D** |
| **oauth** | 100+ (via `oauth.EmailFromContext`) | all dirs | **Cross-cutting** |
| **papertrading** | 8 | `mcp/paper` | **Agent E** |
| **riskguard** | 18 + admin | `mcp/admin`, `mcp/misc` | **Agent C** (with audit/billing) |
| **scheduler** | 8 | `mcp/paper` | **Agent E** (with papertrading) |
| **sectors** | 20 | `mcp/portfolio` | **Agent D** (with money) |
| **ticker** | 8 + alerts + misc | `mcp/alerts`, `mcp/misc` | **Agent B** (with alerts) |
| **users** | 18 admin + 9 misc | `mcp/admin`, `mcp/misc` | **Agent C** (with audit/billing) |
| **watchlist** | 6 root | `mcp/watchlist_tools.go` | **Agent F** |
| **eventsourcing** | admin + misc + paper | `mcp/admin`, `mcp/misc`, `mcp/paper` | **Cross-cutting** |
| **clockport** | (test only) | testutil | **Cross-cutting** |
| **logger** | all (via LoggerPort) | all | **Cross-cutting** |
| **isttz** | (test only) | testutil | **Cross-cutting** |
| **i18n** | (root mcp/) | mcp/ | **Cross-cutting** |
| **decorators** | (middleware) | mcp/middleware | **Cross-cutting** |
| **legaldocs** | (app/) | app/ | **Out-of-scope (not a tool)** |
| **templates** | (app/) | app/ | **Out-of-scope** |
| **aop** | (kc/) | kc/ | **Out-of-scope** |
| **registry** | 18 admin | `mcp/admin` | **Agent C** |
| **telegram** | (app/) | app/ + kc/ | **Out-of-scope (Telegram bot lives in app/)** |

### Proposed fan-out (6 parallel agents)

| Agent | Scope (directories) | Tool count | Modules | Estimated work |
|---|---|---|---|---|
| **A** | `mcp/trade/` | 28 | broker, instruments, ticker, usecases | ~6h |
| **B** | `mcp/alerts/` + `mcp/misc/` | 8 + 9 = 17 | alerts, ticker, riskguard | ~4h |
| **C** | `mcp/admin/` | 18 | audit, billing, riskguard, users, registry, eventsourcing | ~4h |
| **D** | `mcp/portfolio/` + `mcp/analytics/` | 20 + 8 = 28 | broker, money, sectors, usecases | ~6h |
| **E** | `mcp/paper/` | 8 | papertrading, scheduler | ~2h |
| **F** | `mcp/watchlist_tools.go` + `mcp/market_tools.go` + `mcp/tax_tools.go` | 6 + 5 + 1 = 12 | (root files; no specific module) | ~3h |
| **Z (coordinator)** | `mcp/mcp.go` + `mcp/registry.go` + `mcp/common/` callsite-flip + tests | n/a | — | ~3h |

**Total agent-hours: ~28h** (vs the brief's 120-250h estimate — empirically the typed-deps work is already done). Wall-clock ~6h if fanned out in parallel.

---

## §6 — Pre-Flight Empirical Checks Required Before Sprint 5 Execute

Each agent's brief must perform a 10-minute pre-flight before any source change:

1. **Confirm starting tool count** = 111 via `go test ./mcp/... -count=1` + `/healthz total_available` from a locally-compiled binary.
2. **Verify the signature-flip target file count**: `grep -c '^func (\\*[A-Z][a-zA-Z0-9]+) Handler(manager \\*kc\\.Manager)' <scope>/*.go` returns the expected count per agent's scope.
3. **Verify no `Handler(manager *kc.Manager)` variants** exist (we confirmed 0 at HEAD `2b96ef6` — re-check at dispatch time in case any new tools land).
4. **Check for inline `handler := common.NewToolHandler(manager)` callsites**: every Handler body that uses `handler.X` must have this line; the migration drops it. Empirically there are 100+ matching the standard pattern.
5. **For any `manager.X()` or `handler.Manager().X()` ref in scope**: classify per §2 patterns. If new manager.X() refs have appeared (drift since HEAD), document them and ask orchestrator before proceeding.

---

## §7 — Recommended Sequencing

### Phase A — Pilot (single agent, sequential, ~3h)

1. **Agent F first**: `watchlist_tools.go` (6 tools, root-mcp, no algo2go-module subdir, fewest cross-references). Establishes the pattern + validates the signature-flip mechanics.
2. **Surface back**: confirm pilot ships green; capture lessons (any unexpected manager.X() refs, test-helper changes needed, etc.).
3. **Add `SessionRegistryProvider` if needed**: if Agent F's pilot reveals the `manager.SessionManager` pattern needs a new Provider port for the misc dir, add it BEFORE Agent B starts.

### Phase B — Fan-out (5-6 parallel agents, ~6h wall-clock)

After pilot ships, dispatch Agents A through E in parallel against `mcp/trade/`, `mcp/alerts+misc/`, `mcp/admin/`, `mcp/portfolio+analytics/`, `mcp/paper/`. Each works in its own scope; no shared file edits.

### Phase C — Coordinator (Agent Z, sequential after fan-out, ~3h)

After all per-dir flips land:
1. Flip the registry callsite at `mcp/mcp.go` / `mcp/registry.go` (single mass-edit).
2. Update `testutil/kcfixture/` to expose `OptDeps()` helper.
3. Mass-update tool-test fixtures across `mcp/*_test.go`.
4. Drop `NewToolHandler(manager)` from `mcp/common/handler_deps.go` (or keep as the Deps composition path).
5. Run full WSL2 gates: build + `GOWORK=off` build + vet + tests + tool count = 111.

### Phase D — Cleanup (optional, ~2h)

- Remove `(h *ToolHandler).Manager()` accessor (no production refs after the flip).
- Drop `ToolHandler.manager` field (no consumers).
- Drop the residual 5 manager-coupling refs by routing through Deps (admin escape hatches + misc session refs).

---

## §8 — What This PREP Does NOT Cover

- **Implementation execution**: this is READ-ONLY survey. No source touched.
- **Per-agent detailed file lists**: each agent's brief should pull the latest empirical state at dispatch time (re-grep at HEAD).
- **Test-fixture analysis**: tools tests touch `testutil/kcfixture` which composes Manager + Deps; needs separate empirical pass per agent before they start.
- **CI/CD impact**: the signature flip is uniform; CI should pass on the first PR per agent if their scope's gates green. Coordinator must batch the rollout to avoid a half-flipped state at HEAD.
- **Production deploy**: `/healthz total_available=111` must hold throughout. Each agent's commits should be reviewable independently; coordinator ships the registry-callsite flip as the atomic switch.

---

## §9 — Big-Picture Takeaway

The original god-object inventory (commit `7c21e7d`) and Day-1 ops blocker survey (commit `f81c91b`) estimated Sprint 5 at 120-250h. Empirical pre-check finds:

- **Tool.Handler implementations**: 111 (not 128) — matches production
- **Already-typed-deps-routed**: 105 of 111 (95%+)
- **Residual `manager.X()` couplings**: 6 (with documented rationale for each)
- **Migration target (Pattern D.2 signature flip)**: mechanical at ~28 agent-hours

The codebase's `ToolHandlerDeps` infrastructure (27 typed Provider ports + 5-context Investment-K builder pattern) is the **already-shipped substantive decomposition work**. Sprint 5 is the closing signature-flip pass that retires the legacy `*kc.Manager` parameter type. It's the **30%-progress-bar finishing move**, not a 120-250h architectural rework.

This is consistent with the meta-pattern surfaced across the B-series halts + Sprint 2-a + Sprint 3 Option-a: **the codebase is materially more decomposed than the original inventory estimated.** The Anchor 6 PR 6.4 / Wave D Package 7c / Phase 3a Batch 6 work-streams shipped the Provider-interface discipline + ToolHandlerDeps typed-ports surface, leaving Sprint 5 with only the cosmetic signature flip.

**Recommendation**: dispatch Agent F pilot first (~3h, sequential). After it ships green, fan out Agents A-E in parallel (~6h wall-clock), then Agent Z coordinator (~3h). Total: ~12h wall-clock, ~28h agent-hours, NOT 120-250h.

---

## §10 — Sources Cited / Probes Run

All probes empirical at HEAD `2b96ef6`, 2026-05-11 IST.

- `grep -rEn '^func \(\*[A-Z][a-zA-Z0-9]+\) Handler\(manager \*kc\.Manager\)' --include='*.go' mcp/` → 99 matches in subdirs + 12 in root files = 111 total
- `grep -rEn 'common\.NewToolHandler\(manager\)' --include='*.go' mcp/` → 100 production callsites
- `grep -rEn 'manager\.[A-Z]' --include='*.go' mcp/admin mcp/alerts mcp/analytics mcp/misc mcp/paper mcp/portfolio mcp/trade` → 6 production refs
- `grep -rEn 'h\.manager\.[A-Z]' --include='*.go' mcp/` → 0 production refs (10 docstring mentions)
- `grep -rEn 'handler\.Manager\(\)\.[A-Z]' --include='*.go' mcp/` → 1 production ref (`pretrade_tool.go:159`)
- `/healthz total_available` (deployed binary) = 111 (cross-reference: matches handler count exactly)
- `mcp/common/handler_deps.go:30-71` — empirical reading of the 27-port `ToolHandlerDeps` struct
- `kc/manager_interfaces.go:239-263` — 14 Provider compile-time assertions for `*Manager`
- `kc/manager_struct.go` field declarations — confirmed B-series outcomes (`SessionManager`, `SessionSigner`, `ManagedSessionSvc` exported per PRs B1/B2/B4)
- Per-dir module-import grep (§5) — algo2go module ownership map

---

**End of PREP.**
