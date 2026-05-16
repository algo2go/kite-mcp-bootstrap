package app

import (
	"github.com/algo2go/kite-mcp-bootstrap/kc"
	"github.com/algo2go/kite-mcp-oauth"
)

// registryAdapter bridges a registry store to oauth.KeyRegistry.
//
// Phase 3a kc/-side close-out: store is typed as kc.RegistryStoreInterface
// port. The three methods called below — HasEntries, GetByEmail,
// GetByAPIKey — are all on the interface. *registry.Store satisfies it
// structurally so the construction site at app/app.go can pass
// kcManager.RegistryStore() instead of RegistryStoreConcrete().
type registryAdapter struct {
	store kc.RegistryStoreInterface
}

func (a *registryAdapter) HasEntries() bool {
	return a.store.HasEntries()
}

func (a *registryAdapter) GetByEmail(email string) (*oauth.RegistryEntry, bool) {
	reg, found := a.store.GetByEmail(email)
	if !found {
		return nil, false
	}
	return &oauth.RegistryEntry{
		APIKey:       reg.APIKey,
		APISecret:    reg.APISecret,
		RegisteredBy: reg.RegisteredBy,
	}, true
}

func (a *registryAdapter) GetSecretByAPIKey(apiKey string) (apiSecret string, ok bool) {
	reg, found := a.store.GetByAPIKey(apiKey)
	if !found {
		return "", false
	}
	return reg.APISecret, true
}
