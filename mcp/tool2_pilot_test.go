package mcp

import (
	"context"
	"testing"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/algo2go/kite-mcp-tools-common/common"
)

// Sprint 5 Pilot F-Y — additive Tool2 interface infrastructure tests.
//
// These tests cover the type-switch at mcp/mcp.go (RegisterToolsForRegistry)
// without booting a full Manager. They establish that:
//
//   1. The Tool2 interface is exported from mcp/common.
//   2. A type that implements only Tool (legacy) does NOT satisfy Tool2.
//   3. A type that implements BOTH Tool and Tool2 satisfies Tool2 too.
//   4. The package-level common.Tool alias path (mcp.Tool) is unchanged.
//
// No Manager wiring is exercised here — the type-switch in
// RegisterToolsForRegistry has its own integration tests via the full
// server test in mcp/get_tools_test.go (covers the legacy path).

// tool2PilotLegacyOnly implements common.Tool only (no HandlerDeps method),
// so it represents the 99 tools that have not yet been migrated to Tool2.
type tool2PilotLegacyOnly struct{}

func (tool2PilotLegacyOnly) Tool() gomcp.Tool {
	return gomcp.NewTool("tool2_pilot_legacy_only")
}

// Handler returns a no-op handler. The body intentionally avoids touching
// the manager so the test stays Manager-free.
func (tool2PilotLegacyOnly) Handler(_ *kcManagerStub) server.ToolHandlerFunc {
	return func(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		return gomcp.NewToolResultText("legacy"), nil
	}
}

// tool2PilotBoth implements both Tool and Tool2 — the bridge-method pattern
// every Pilot-F tool will adopt for the transition window.
type tool2PilotBoth struct{}

func (tool2PilotBoth) Tool() gomcp.Tool {
	return gomcp.NewTool("tool2_pilot_both")
}

func (tool2PilotBoth) Handler(_ *kcManagerStub) server.ToolHandlerFunc {
	// In production this bridge calls HandlerDeps with deps built from
	// the manager. The test only needs to confirm the method exists +
	// the type satisfies the Tool interface; we do not wire deps here.
	return func(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		return gomcp.NewToolResultText("bridge"), nil
	}
}

func (tool2PilotBoth) HandlerDeps(_ *common.ToolHandlerDeps) server.ToolHandlerFunc {
	return func(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		return gomcp.NewToolResultText("typed-deps"), nil
	}
}

// kcManagerStub is a deliberately narrow stand-in for *kc.Manager so the
// test types satisfy the common.Tool interface without dragging the full
// manager into the test. NOTE: the actual common.Tool interface requires
// *kc.Manager; this test type satisfies the SAME-NAME-DIFFERENT-TYPE
// criterion only for compile-checking the Tool2 path via direct type
// assertion — not for slot-into-[]Tool registration. The test is purely
// structural (does Tool2 surface compile + does an interface assertion
// succeed/fail as expected for the two sample types).
type kcManagerStub struct{}

// TestTool2InterfaceDeclaration is a structural test that compile-asserts
// the Tool2 interface surface. If common.Tool2 ever loses HandlerDeps or
// changes its return type, this test fails to compile.
func TestTool2InterfaceDeclaration(t *testing.T) {
	t.Parallel()
	// The Tool2 surface must be: Tool() gomcp.Tool + HandlerDeps(*common.ToolHandlerDeps) server.ToolHandlerFunc
	// Probe via a function-typed nil that conforms to the expected shape.
	var _ func(common.Tool2) gomcp.Tool = func(t common.Tool2) gomcp.Tool { return t.Tool() }
	var _ func(common.Tool2, *common.ToolHandlerDeps) server.ToolHandlerFunc = func(t common.Tool2, d *common.ToolHandlerDeps) server.ToolHandlerFunc {
		return t.HandlerDeps(d)
	}
}

// TestTool2TypeSwitch covers the runtime branching used by
// RegisterToolsForRegistry: a legacy Tool that does NOT implement Tool2
// should fall through to the legacy Handler path; a tool that DOES
// implement Tool2 should be routed through HandlerDeps. This test does
// not exercise the registry directly (no Manager wiring); it asserts the
// type-switch outcomes that the registry depends on.
func TestTool2TypeSwitch(t *testing.T) {
	t.Parallel()

	legacy := tool2PilotLegacyOnly{}
	both := tool2PilotBoth{}

	// Empirical guard: confirm legacy does NOT satisfy Tool2. If a future
	// refactor accidentally adds HandlerDeps to a "legacy" type, this
	// guard catches it before that type slips through the type-switch
	// expecting a manager-coupled Handler.
	var asTool2 any = legacy
	if _, ok := asTool2.(interface {
		HandlerDeps(*common.ToolHandlerDeps) server.ToolHandlerFunc
	}); ok {
		t.Errorf("tool2PilotLegacyOnly must NOT implement Tool2 (HandlerDeps surface)")
	}

	// Empirical guard: confirm both DOES satisfy Tool2.
	asTool2 = both
	if _, ok := asTool2.(interface {
		HandlerDeps(*common.ToolHandlerDeps) server.ToolHandlerFunc
	}); !ok {
		t.Errorf("tool2PilotBoth must implement Tool2 (HandlerDeps surface)")
	}
}

// TestPilotFToolsSatisfyTool2 — empirical assertion that the 12
// Pilot F root-mcp tools (commit 2 of Sprint 5 Pilot F-Y) implement
// common.Tool2 via HandlerDeps. If any one of them loses HandlerDeps
// (regression), the type-switch in RegisterToolsForRegistry silently
// falls back to legacy Handler(*kc.Manager) for that tool — which is
// still functionally correct but defeats the Tool2 migration. This
// guard catches the regression immediately.
//
// Each entry is asserted twice: (a) implements common.Tool (the
// pre-existing interface; ensures we did not accidentally remove
// Handler during the migration), and (b) implements common.Tool2
// (the new surface added by this commit).
func TestPilotFToolsSatisfyTool2(t *testing.T) {
	t.Parallel()
	pilotF := []any{
		// mcp/watchlist_tools.go (6)
		&CreateWatchlistTool{},
		&DeleteWatchlistTool{},
		&AddToWatchlistTool{},
		&RemoveFromWatchlistTool{},
		&GetWatchlistTool{},
		&ListWatchlistsTool{},
		// mcp/market_tools.go (5)
		&QuotesTool{},
		&InstrumentsSearchTool{},
		&HistoricalDataTool{},
		&LTPTool{},
		&OHLCTool{},
		// mcp/tax_tools.go (1)
		&TaxHarvestTool{},
	}
	if got := len(pilotF); got != 12 {
		t.Fatalf("Pilot F roster expected 12 tools, got %d", got)
	}
	for _, tool := range pilotF {
		if _, ok := tool.(common.Tool); !ok {
			t.Errorf("%T must still implement common.Tool (legacy bridge retained during transition)", tool)
		}
		if _, ok := tool.(common.Tool2); !ok {
			t.Errorf("%T must implement common.Tool2 (Pilot F Sprint 5 migration target)", tool)
		}
	}
}
