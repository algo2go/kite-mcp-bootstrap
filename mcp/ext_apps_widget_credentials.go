package mcp

import (
	"context"
	"time"

	"github.com/algo2go/kite-mcp-audit"
)

// credentialsData returns current credential metadata for the rotation widget.
// Only the API key is surfaced (masked) — the secret is never read back into
// the widget. last_updated comes from KiteCredentialEntry.StoredAt which is
// set on every Set() call in the credential store.
func credentialsData(_ context.Context, manager extAppManagerPort, _ *audit.Store, email string) any {
	resp := map[string]any{
		"credentials_registered": false,
		"api_key_masked":         "",
		"last_updated":           "",
	}
	store := manager.CredentialStore()
	if store == nil {
		return resp
	}
	entry, ok := store.Get(email)
	if !ok {
		return resp
	}
	resp["credentials_registered"] = true
	resp["api_key_masked"] = maskAPIKey(entry.APIKey)
	if !entry.StoredAt.IsZero() {
		resp["last_updated"] = entry.StoredAt.Format(time.RFC3339)
	}
	return resp
}
