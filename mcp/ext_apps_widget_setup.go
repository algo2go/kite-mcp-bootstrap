package mcp

import (
	"context"

	"github.com/algo2go/kite-mcp-audit"
	"github.com/algo2go/kite-mcp-bootstrap/mcp/paper"
)

// setupData returns onboarding state for the setup checklist widget.
// Step 1 (credentials registered) is hydrated from the credential store.
// Step 2 (IP whitelisted) can't be verified independently — it's confirmed
// indirectly when Step 3 (test_ip_whitelist tool) passes. Until then the
// widget shows Step 2 as unverified.
func setupData(_ context.Context, manager extAppManagerPort, _ *audit.Store, email string) any {
	credsRegistered := false
	apiKeyMasked := ""
	if store := manager.CredentialStore(); store != nil {
		if entry, ok := store.Get(email); ok {
			credsRegistered = true
			apiKeyMasked = maskAPIKey(entry.APIKey)
		}
	}
	return map[string]any{
		"egress_ip":              paper.SetupStaticEgressIP,
		"credentials_registered": credsRegistered,
		"api_key_masked":         apiKeyMasked,
		// ready_to_trade currently mirrors credentials_registered — Step 3
		// connectivity result lives in the browser, not server-side state.
		"ready_to_trade": credsRegistered,
	}
}

// maskAPIKey returns a masked form of a Kite API key: first 3 chars, stars,
// last 3 chars, e.g. "4c0****...3b7". Returns empty string for short/empty input.
func maskAPIKey(apiKey string) string {
	if len(apiKey) < 7 {
		return ""
	}
	return apiKey[:3] + "****..." + apiKey[len(apiKey)-3:]
}
