package app

import (
	"github.com/algo2go/kite-mcp-kc"
	"github.com/algo2go/kite-mcp-instruments"
)

// riskguardLTPAdapter bridges paperLTPAdapter (kite-style "EXCHANGE:SYMBOL"
// argument) to riskguard.LTPLookup (separate exchange + tradingsymbol
// arguments). PR-C uses this to plumb Kite live quotes into the SEBI
// OTR band check. Reuses paperLTPAdapter.GetLTP under the hood — same
// active-session iteration, same client lookup, same fallback semantics.
type riskguardLTPAdapter struct {
	manager *kc.Manager
}

// GetLTP looks up the last-traded price for one instrument. Returns
// (price, true) on success, (0, false) on any failure (no active
// sessions, broker unavailable, instrument not quoted). The OTR band
// check fails open on (_, false), which is the intended SEBI-
// conservative behaviour (don't block valid orders on missing oracle
// data).
func (a *riskguardLTPAdapter) GetLTP(exchange, tradingsymbol string) (float64, bool) {
	if a.manager == nil || exchange == "" || tradingsymbol == "" {
		return 0, false
	}
	key := exchange + ":" + tradingsymbol
	bridge := &paperLTPAdapter{manager: a.manager}
	ltps, err := bridge.GetLTP(key)
	if err != nil {
		return 0, false
	}
	p, ok := ltps[key]
	if !ok || p <= 0 {
		return 0, false
	}
	return p, true
}

// instrumentsFreezeAdapter wraps instruments.Manager to implement riskguard.FreezeQuantityLookup.
type instrumentsFreezeAdapter struct {
	mgr *instruments.Manager
}

func (a *instrumentsFreezeAdapter) GetFreezeQuantity(exchange, tradingsymbol string) (uint32, bool) {
	inst, err := a.mgr.GetByTradingsymbol(exchange, tradingsymbol)
	if err != nil {
		return 0, false
	}
	return inst.FreezeQuantity, inst.FreezeQuantity > 0
}
