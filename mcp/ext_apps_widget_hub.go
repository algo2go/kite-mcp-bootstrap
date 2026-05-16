package mcp

import (
	"context"
	"time"

	"github.com/algo2go/kite-mcp-bootstrap/kc"
	"github.com/algo2go/kite-mcp-audit"
)

// hubData returns account status, quick stats, and external URL for the hub widget.
func hubData(_ context.Context, manager extAppManagerPort, auditStore *audit.Store, email string) any {
	_, hasCreds := manager.CredentialStore().Get(email)

	kiteConnected := false
	if entry, ok := manager.TokenStore().Get(email); ok {
		kiteConnected = !kc.IsKiteTokenExpired(entry.StoredAt)
	}

	paperOn := false
	if engine := manager.PaperEngine(); engine != nil {
		paperOn = engine.IsEnabled(email)
	}

	alertCount := 0
	if manager.AlertStore() != nil {
		for _, a := range manager.AlertStore().List(email) {
			if !a.Triggered {
				alertCount++
			}
		}
	}

	toolCallsToday := 0
	if auditStore != nil {
		since := time.Now().Truncate(24 * time.Hour)
		if stats, err := auditStore.GetStats(email, since, "", false); err == nil {
			toolCallsToday = stats.TotalCalls
		}
	}

	externalURL := manager.ExternalURL()
	if externalURL == "" {
		externalURL = "https://kite-mcp-server.fly.dev"
	}

	return map[string]any{
		"email":            email,
		"kite_connected":   kiteConnected,
		"credentials_set":  hasCreds,
		"paper_mode":       paperOn,
		"active_alerts":    alertCount,
		"tool_calls_today": toolCallsToday,
		"external_url":     externalURL,
	}
}
