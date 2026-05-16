package app

import (
	"time"

	"github.com/algo2go/kite-mcp-bootstrap/kc"
)

// briefingTokenAdapter bridges a token store to alerts.TokenChecker.
//
// Phase 3a kc/-side close-out: store is typed as the kc.TokenStoreInterface
// port rather than the concrete *kc.KiteTokenStore. The two methods used
// by GetToken — Get(email) — are part of the interface, so narrowing
// the field eliminates a Concrete-pattern leak at this construction site
// without behaviour change. The concrete *kc.KiteTokenStore satisfies
// the interface structurally so existing struct-literal sites (tests,
// wire.go's briefingSvc construction) keep compiling under
// implicit-conversion.
type briefingTokenAdapter struct {
	store kc.TokenStoreInterface
}

func (a *briefingTokenAdapter) GetToken(email string) (string, time.Time, bool) {
	entry, ok := a.store.Get(email)
	if !ok {
		return "", time.Time{}, false
	}
	return entry.AccessToken, entry.StoredAt, true
}

func (a *briefingTokenAdapter) IsExpired(storedAt time.Time) bool {
	return kc.IsKiteTokenExpired(storedAt)
}

// briefingCredAdapter bridges kc.Manager to alerts.CredentialGetter.
type briefingCredAdapter struct {
	manager *kc.Manager
}

func (a *briefingCredAdapter) GetAPIKey(email string) string {
	return a.manager.GetAPIKeyForEmail(email)
}
