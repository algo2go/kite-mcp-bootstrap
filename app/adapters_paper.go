package app

import (
	"fmt"

	"github.com/algo2go/kite-mcp-bootstrap/kc"
)

// paperLTPAdapter bridges kc.Manager to papertrading.LTPProvider by using
// any active session's Kite client for read-only LTP lookups.
type paperLTPAdapter struct {
	manager *kc.Manager
}

func (a *paperLTPAdapter) GetLTP(instruments ...string) (map[string]float64, error) {
	sessions := a.manager.SessionManager().ListActiveSessions()
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no active Kite sessions for LTP lookup")
	}
	for _, sess := range sessions {
		data, ok := sess.Data.(*kc.KiteSessionData)
		if !ok || data == nil || data.Kite == nil {
			continue
		}
		ltps, err := data.Kite.GetLTP(instruments...)
		if err != nil {
			continue
		}
		result := make(map[string]float64, len(ltps))
		for k, v := range ltps {
			result[k] = v.LastPrice
		}
		return result, nil
	}
	return nil, fmt.Errorf("no Kite client available for LTP")
}
