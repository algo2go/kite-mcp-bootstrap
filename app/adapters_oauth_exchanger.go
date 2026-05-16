package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/algo2go/kite-mcp-broker"
	"github.com/algo2go/kite-mcp-kc"
	"github.com/algo2go/kite-mcp-audit"
	"github.com/algo2go/kite-mcp-cqrs"
	logport "github.com/algo2go/kite-mcp-logger"
	"github.com/algo2go/kite-mcp-usecases"
)

// kiteExchangerAdapter exchanges a Kite request_token for user identity.
//
// Every WRITE in this adapter dispatches through the CommandBus instead of
// touching stores directly — this keeps the CQRS contract uniform across
// the codebase (every mutation hits LoggingMiddleware uniformly). The
// stored *kc.* references are kept only for READS (GetCredentials,
// GetSecretByAPIKey) which are cheap and lock-free.
//
// commandBus is a structural invariant: it is NEVER nil at use time. The
// production wire-up sets it from kcManager.CommandBus(); tests that
// build a struct literal without one trigger ensureBus() on first use,
// which constructs a local InMemoryBus with the same handlers the manager
// would have wired. The adapter therefore has a single dispatch path —
// no "fallback to raw store write" gate.
type kiteExchangerAdapter struct {
	apiKey    string
	apiSecret string
	// Phase 3a kc/-side close-out: the four store fields below are typed
	// as kc-package interfaces (TokenStoreInterface, CredentialStoreInterface,
	// users.UserStore via the kc.UserStoreInterface alias path here is
	// retained as *users.Store because users.Store has a few non-interface
	// methods the local-bus path uses indirectly via localUserProvisioner —
	// see adapters_local_bus.go's narrowed port story below) where the
	// methods used by both direct reads (GetCredentials, GetSecretByAPIKey)
	// AND the test-local-bus handler backing (oauthBridgeStores → local*
	// adapters) are all on the kc-level interface surface. *kc.KiteTokenStore
	// / *kc.KiteCredentialStore / *registry.Store / *users.Store satisfy
	// these interfaces structurally so app/app.go's struct-literal
	// construction site can pass kcManager.TokenStore() / .CredentialStore()
	// / .RegistryStore() / .UserStore() (port accessors) instead of the
	// *Concrete() siblings.
	tokenStore      kc.TokenStoreInterface      // read paths AND test-local-bus handler backing
	credentialStore kc.CredentialStoreInterface // read paths AND test-local-bus handler backing
	registryStore   kc.RegistryStoreInterface   // test-local-bus handler backing
	userStore       kc.UserStoreInterface       // test-local-bus handler backing
	// Wave D Phase 3 Package 7c-4b: logger field carries the
	// kc/logger.Logger port. Constructor sites (adapters_local_bus.go)
	// take *slog.Logger params and wrap via logport.NewSlog.
	logger        logport.Logger
	authenticator broker.Authenticator
	commandBus    cqrs.CommandBus // never nil at use time — see ensureBus
	busOnce       sync.Once
}

// ensureBus guarantees a.commandBus is non-nil before any Dispatch call.
// In production, app/app.go wires kcManager.CommandBus() at struct-literal
// time so this is a no-op (commandBus already non-nil). In tests that
// build a struct literal without a manager, this constructs an in-process
// bus with the same six OAuth-bridge handlers the manager would have
// registered, backed by whatever stores the test put on the adapter.
//
// Rationale: the adapter MUST always go through Dispatch, never a raw
// store write — that's the CQRS invariant. We satisfy it by ensuring
// every code path has a real bus, not by gating writes on a nil check.
func (a *kiteExchangerAdapter) ensureBus() {
	a.busOnce.Do(func() {
		if a.commandBus != nil {
			return
		}
		// newLocalOAuthBridgeBus takes *slog.Logger (Package 7c keeps
		// signature stable for its other callers). AsSlog unwraps the
		// port back to slog at this single boundary.
		a.commandBus = newLocalOAuthBridgeBus(logport.AsSlog(a.logger), oauthBridgeStores{
			Users:       a.userStore,
			Tokens:      a.tokenStore,
			Credentials: a.credentialStore,
			Registry:    a.registryStore,
		})
	})
}

// provisionUser auto-provisions a user on first OAuth login and checks status.
// Returns an error if the user is suspended or offboarded.
//
// Single dispatch path: ensureBus() guarantees a non-nil bus, then we
// dispatch ProvisionUserOnLoginCommand. The use case in
// kc/usecases/oauth_bridge_usecases.go owns the suspended/offboarded →
// error mapping.
//
// E1+E4: errors wrap the upstream sentinel via %w so caller-side
// errors.Is checks still match, AND the email is hashed (audit.HashEmail
// — same canonical form the consent log uses) before being embedded
// in the message. Plaintext emails in error strings leak through every
// log layer the error touches; the hash gives operators correlation
// power without the PII exposure.
func (a *kiteExchangerAdapter) provisionUser(email, kiteUID, displayName string) error {
	email = strings.ToLower(email)
	a.ensureBus()
	err := a.commandBus.Dispatch(context.Background(), cqrs.ProvisionUserOnLoginCommand{
		Email:       email,
		KiteUID:     kiteUID,
		DisplayName: displayName,
	})
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, usecases.ErrUserSuspended):
		return fmt.Errorf("user account is suspended (email_hash=%s): %w", audit.HashEmail(email), usecases.ErrUserSuspended)
	case errors.Is(err, usecases.ErrUserOffboarded):
		return fmt.Errorf("user account has been offboarded (email_hash=%s): %w", audit.HashEmail(email), usecases.ErrUserOffboarded)
	default:
		return err
	}
}

func (a *kiteExchangerAdapter) ExchangeRequestToken(requestToken string) (string, error) {
	result, err := a.authenticator.ExchangeToken(a.apiKey, a.apiSecret, requestToken)
	if err != nil {
		return "", fmt.Errorf("kite generate session: %w", err)
	}

	email := result.Email
	if email == "" {
		email = result.UserID
	}

	// Auto-provision user and check status (dispatched via bus, with
	// direct-store fallback when no bus is wired — see provisionUser).
	if err := a.provisionUser(email, result.UserID, result.UserName); err != nil {
		return "", err
	}

	a.logger.Debug(context.Background(), "Kite token exchange successful", "email", email, "user_id", result.UserID)

	// Token cache + registry-stamp writes — single dispatch path via the
	// bus. ensureBus() above already guaranteed non-nil; provisionUser
	// called it for us.
	if dispErr := a.commandBus.Dispatch(context.Background(), cqrs.CacheKiteAccessTokenCommand{
		Email:       email,
		AccessToken: result.AccessToken,
		UserID:      result.UserID,
		UserName:    result.UserName,
	}); dispErr != nil {
		a.logger.Error(context.Background(), "Failed to dispatch CacheKiteAccessTokenCommand", dispErr, "email", email)
	}
	if a.apiKey != "" {
		if dispErr := a.commandBus.Dispatch(context.Background(), cqrs.SyncRegistryAfterLoginCommand{
			Email:        email,
			APIKey:       a.apiKey,
			AutoRegister: false,
		}); dispErr != nil {
			a.logger.Debug(context.Background(), "SyncRegistryAfterLoginCommand global-stamp dispatch failed", "error", dispErr)
		}
	}

	return email, nil
}

func (a *kiteExchangerAdapter) ExchangeWithCredentials(requestToken, apiKey, apiSecret string) (string, error) {
	result, err := a.authenticator.ExchangeToken(apiKey, apiSecret, requestToken)
	if err != nil {
		return "", fmt.Errorf("kite generate session with per-user credentials: %w", err)
	}

	email := result.Email
	if email == "" {
		email = result.UserID
	}

	// Auto-provision user and check status (dispatched via bus).
	if err := a.provisionUser(email, result.UserID, result.UserName); err != nil {
		return "", err
	}

	a.logger.Debug(context.Background(), "Kite token exchange (per-user credentials) successful", "email", email, "user_id", result.UserID)
	lowerEmail := strings.ToLower(email)

	// Three writes in sequence: token cache, credential store, registry sync.
	// Each dispatched as a separate command. ensureBus() in provisionUser
	// already guaranteed a.commandBus is non-nil.
	if dispErr := a.commandBus.Dispatch(context.Background(), cqrs.CacheKiteAccessTokenCommand{
		Email:       lowerEmail,
		AccessToken: result.AccessToken,
		UserID:      result.UserID,
		UserName:    result.UserName,
	}); dispErr != nil {
		a.logger.Error(context.Background(), "Failed to dispatch CacheKiteAccessTokenCommand", dispErr, "email", lowerEmail)
	}
	if dispErr := a.commandBus.Dispatch(context.Background(), cqrs.StoreUserKiteCredentialsCommand{
		Email:     lowerEmail,
		APIKey:    apiKey,
		APISecret: apiSecret,
	}); dispErr != nil {
		a.logger.Error(context.Background(), "Failed to dispatch StoreUserKiteCredentialsCommand", dispErr, "email", lowerEmail)
	}
	if dispErr := a.commandBus.Dispatch(context.Background(), cqrs.SyncRegistryAfterLoginCommand{
		Email:        lowerEmail,
		APIKey:       apiKey,
		APISecret:    apiSecret,
		Label:        "Self-provisioned",
		AutoRegister: true,
	}); dispErr != nil {
		a.logger.Error(context.Background(), "Failed to dispatch SyncRegistryAfterLoginCommand", dispErr, "email", lowerEmail)
	}

	return email, nil
}

func (a *kiteExchangerAdapter) GetCredentials(email string) (string, string, bool) {
	email = strings.ToLower(email)
	entry, ok := a.credentialStore.Get(email)
	if !ok {
		// Fall back to global credentials if available
		if a.apiKey != "" && a.apiSecret != "" {
			return a.apiKey, a.apiSecret, true
		}
		return "", "", false
	}
	return entry.APIKey, entry.APISecret, true
}

func (a *kiteExchangerAdapter) GetSecretByAPIKey(apiKey string) (string, bool) {
	return a.credentialStore.GetSecretByAPIKey(apiKey)
}
