package app

import (
	"context"
	"sync"
	"time"

	"github.com/algo2go/kite-mcp-alerts"
	"github.com/algo2go/kite-mcp-cqrs"
	logport "github.com/algo2go/kite-mcp-logger"
	"github.com/algo2go/kite-mcp-oauth"
)

// clientPersisterAdapter bridges alerts.DB to oauth.ClientPersister.
//
// Reads (LoadClients) bypass the bus — they're idempotent queries with no
// state change. Writes (SaveClient, DeleteClient) dispatch through the
// CommandBus so every OAuth-client mutation hits LoggingMiddleware, same
// as every other write in the codebase.
//
// commandBus is a structural invariant: NEVER nil at use time.
// ensureBus() lazily constructs a local InMemoryBus when none was wired
// (e.g. unit tests that build a struct literal). No "bus or raw write"
// gate — every code path goes through Dispatch.
type clientPersisterAdapter struct {
	db         *alerts.DB
	commandBus cqrs.CommandBus
	// Wave D Phase 3 Package 7c-4b: see kiteExchangerAdapter for the
	// logport.Logger port migration rationale.
	logger  logport.Logger
	busOnce sync.Once
}

func (a *clientPersisterAdapter) ensureBus() {
	a.busOnce.Do(func() {
		if a.commandBus != nil {
			return
		}
		// AsSlog: see kiteExchangerAdapter.ensureBus rationale.
		a.commandBus = newLocalOAuthClientBus(logport.AsSlog(a.logger), a.db)
	})
}

// SaveClient dispatches a SaveOAuthClientCommand. ensureBus() guarantees
// a non-nil bus first; production wires kcManager.CommandBus() directly,
// tests get a local InMemoryBus that hits the same use case.
func (a *clientPersisterAdapter) SaveClient(clientID, clientSecret, redirectURIsJSON, clientName string, createdAt time.Time, isKiteKey bool) error {
	a.ensureBus()
	return a.commandBus.Dispatch(context.Background(), cqrs.SaveOAuthClientCommand{
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		RedirectURIsJSON: redirectURIsJSON,
		ClientName:       clientName,
		CreatedAtUnix:    createdAt.UnixNano(),
		IsKiteAPIKey:     isKiteKey,
	})
}

func (a *clientPersisterAdapter) LoadClients() ([]*oauth.ClientLoadEntry, error) {
	entries, err := a.db.LoadClients()
	if err != nil {
		return nil, err
	}
	result := make([]*oauth.ClientLoadEntry, len(entries))
	for i, e := range entries {
		result[i] = &oauth.ClientLoadEntry{
			ClientID:     e.ClientID,
			ClientSecret: e.ClientSecret,
			RedirectURIs: e.RedirectURIs,
			ClientName:   e.ClientName,
			CreatedAt:    e.CreatedAt,
			IsKiteAPIKey: e.IsKiteAPIKey,
		}
	}
	return result, nil
}

// DeleteClient dispatches a DeleteOAuthClientCommand.
func (a *clientPersisterAdapter) DeleteClient(clientID string) error {
	a.ensureBus()
	return a.commandBus.Dispatch(context.Background(), cqrs.DeleteOAuthClientCommand{
		ClientID: clientID,
	})
}
