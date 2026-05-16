package mcp

import (
	"context"
	"sync"

	"github.com/algo2go/kite-mcp-audit"
)

// paperData fetches paper trading status, holdings, and positions for the widget.
func paperData(_ context.Context, manager extAppManagerPort, _ *audit.Store, email string) any {
	engine := manager.PaperEngine()
	if engine == nil {
		return map[string]any{"status": map[string]any{"enabled": false, "message": "Paper trading engine not configured."}}
	}

	status, err := engine.Status(email)
	if err != nil {
		return map[string]any{"error": "Failed to get paper status: " + err.Error()}
	}

	enabled, _ := status["enabled"].(bool)
	if !enabled {
		return map[string]any{"status": status}
	}

	// Fetch holdings and positions in parallel.
	var holdings, positions any
	var holdingsErr, positionsErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); holdings, holdingsErr = engine.GetHoldings(email) }()
	go func() {
		defer wg.Done()
		posResult, err := engine.GetPositions(email)
		if err != nil {
			positionsErr = err
			return
		}
		// GetPositions returns map[string]any{"net":..., "day":...}; extract net.
		if posMap, ok := posResult.(map[string]any); ok {
			if net, ok := posMap["net"]; ok {
				positions = net
			} else {
				positions = posResult
			}
		} else {
			positions = posResult
		}
	}()
	wg.Wait()

	if holdingsErr != nil {
		return map[string]any{"error": "Failed to get paper holdings: " + holdingsErr.Error()}
	}
	if positionsErr != nil {
		return map[string]any{"error": "Failed to get paper positions: " + positionsErr.Error()}
	}

	return map[string]any{
		"status":    status,
		"holdings":  holdings,
		"positions": positions,
	}
}
