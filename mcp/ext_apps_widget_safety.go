package mcp

import (
	"context"

	"github.com/algo2go/kite-mcp-audit"
)

// safetyData fetches riskguard status and limits for the safety widget.
func safetyData(_ context.Context, manager extAppManagerPort, auditStore *audit.Store, email string) any {
	guard := manager.RiskGuard()
	if guard == nil {
		return map[string]any{
			"enabled": false,
			"message": "RiskGuard is not enabled on this server.",
		}
	}

	status := guard.GetUserStatus(email)
	limits := guard.GetEffectiveLimits(email)

	_, hasToken := manager.TokenStore().Get(email)
	_, hasCreds := manager.CredentialStore().Get(email)

	return map[string]any{
		"enabled": true,
		"status":  status,
		"limits": map[string]any{
			"max_single_order_inr":  limits.MaxSingleOrderINR.Float64(),
			"max_orders_per_day":    limits.MaxOrdersPerDay,
			"max_orders_per_minute": limits.MaxOrdersPerMinute,
			"duplicate_window_secs": limits.DuplicateWindowSecs,
			"max_daily_value_inr":   limits.MaxDailyValueINR.Float64(),
			"auto_freeze_on_limit":  limits.AutoFreezeOnLimitHit,
		},
		"sebi": map[string]any{
			"static_egress_ip": true,
			"session_active":   hasToken,
			"credentials_set":  hasCreds,
			"order_tagging":    true,
			"audit_trail":      auditStore != nil,
		},
	}
}
