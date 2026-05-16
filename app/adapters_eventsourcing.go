package app

import (
	"context"
	"log/slog"

	"github.com/algo2go/kite-mcp-audit"
	"github.com/algo2go/kite-mcp-domain"
	"github.com/algo2go/kite-mcp-eventsourcing"
	logport "github.com/algo2go/kite-mcp-logger"
)

// makeEventPersister returns a domain.Event handler that appends events to the domain audit log.
// This is the production event persistence path — events are written but never read back
// for state reconstitution. The domain_events table serves as an immutable audit trail
// for compliance, debugging, and activity dashboards.
// Each event is stored with the given aggregateType. The aggregate ID is derived from
// the event's fields (e.g. OrderID for orders, AlertID for alerts, Email for users).
//
// PR-D Item 2: deriveEmailHash extracts the user-association field from
// the typed event (Email / AdminEmail) and stores its SHA-256 hex digest
// on StoredEvent.EmailHash. The persisted row carries the hash, never
// the plaintext — the original event payload is JSON-marshalled as-is
// for in-process consumers, but the indexable email_hash column gives
// auditors and the data-portability export a PII-free correlation key.
// Wave D Phase 3 Package 7c-4b: signature retains *slog.Logger for
// backward-compat with existing callers (app/wire.go, tests). The
// internal log path uses kc/logger.Logger via logport.NewSlog so
// the typed-port + ctx threading benefits propagate without breaking
// the public surface. Constructor-shim pattern matches the
// established deprecation approach in cqrs/billing/ops Package 7c.
func makeEventPersister(store *eventsourcing.EventStore, aggregateType string, logger *slog.Logger) func(domain.Event) {
	port := logport.NewSlog(logger)
	return func(e domain.Event) {
		// Event persister has no request ctx in scope; use
		// context.Background() per the helper-function convention.
		ctx := context.Background()
		aggregateID := deriveAggregateID(e)
		payload, err := eventsourcing.MarshalPayload(e)
		if err != nil {
			port.Error(ctx, "Failed to marshal domain event payload", err, "event_type", e.EventType())
			return
		}
		seq, err := store.NextSequence(aggregateID)
		if err != nil {
			port.Error(ctx, "Failed to get next sequence", err, "event_type", e.EventType(), "aggregate", aggregateID)
			return
		}
		if err := store.Append(eventsourcing.StoredEvent{
			AggregateID:   aggregateID,
			AggregateType: aggregateType,
			EventType:     e.EventType(),
			Payload:       payload,
			OccurredAt:    e.OccurredAt(),
			Sequence:      seq,
			EmailHash:     deriveEmailHash(e),
		}); err != nil {
			port.Error(ctx, "Failed to persist domain event", err, "event_type", e.EventType())
		}
	}
}

// deriveEmailHash extracts the user-association field from a typed
// domain.Event and returns its SHA-256 hex digest. Returns "" for
// system events that have no user (GlobalFreezeEvent, etc.).
//
// Centralised here so the persister and any future direct-Append
// callers (use cases that bypass the dispatcher path) produce
// identical hash values.
func deriveEmailHash(e domain.Event) string {
	switch ev := e.(type) {
	case domain.OrderPlacedEvent:
		return audit.HashEmail(ev.Email)
	case domain.OrderModifiedEvent:
		return audit.HashEmail(ev.Email)
	case domain.OrderCancelledEvent:
		return audit.HashEmail(ev.Email)
	case domain.OrderFilledEvent:
		return audit.HashEmail(ev.Email)
	case domain.PositionOpenedEvent:
		return audit.HashEmail(ev.Email)
	case domain.PositionClosedEvent:
		return audit.HashEmail(ev.Email)
	case domain.AlertCreatedEvent:
		return audit.HashEmail(ev.Email)
	case domain.AlertTriggeredEvent:
		return audit.HashEmail(ev.Email)
	case domain.AlertDeletedEvent:
		return audit.HashEmail(ev.Email)
	case domain.UserFrozenEvent:
		return audit.HashEmail(ev.Email)
	case domain.UserSuspendedEvent:
		return audit.HashEmail(ev.Email)
	case domain.RiskLimitBreachedEvent:
		return audit.HashEmail(ev.Email)
	case domain.SessionCreatedEvent:
		return audit.HashEmail(ev.Email)
	// SessionCleared / SessionInvalidated key by session_id only, no
	// email field — empty hash means "session-scoped, not user-scoped".
	case domain.FamilyInvitedEvent:
		// Hash the admin (the data subject doing the inviting). The
		// invited email is also user data but isn't queried-by-user
		// in our schema; if needed a future migration can split.
		return audit.HashEmail(ev.AdminEmail)
	case domain.FamilyMemberRemovedEvent:
		return audit.HashEmail(ev.AdminEmail)
	case domain.WatchlistCreatedEvent:
		return audit.HashEmail(ev.Email)
	case domain.WatchlistDeletedEvent:
		return audit.HashEmail(ev.Email)
	case domain.WatchlistItemAddedEvent:
		return audit.HashEmail(ev.Email)
	case domain.WatchlistItemRemovedEvent:
		return audit.HashEmail(ev.Email)
	case domain.CredentialRegisteredEvent:
		return audit.HashEmail(ev.Email)
	case domain.CredentialRotatedEvent:
		return audit.HashEmail(ev.Email)
	case domain.CredentialRevokedEvent:
		return audit.HashEmail(ev.Email)
	case domain.ConsentWithdrawnEvent:
		// Already pre-hashed by the use case; pass through if non-empty.
		if ev.EmailHash != "" {
			return ev.EmailHash
		}
		return audit.HashEmail(ev.Email)
	case domain.TierChangedEvent:
		return audit.HashEmail(ev.UserEmail)
	case domain.AnomalyBaselineSnapshottedEvent:
		return audit.HashEmail(ev.UserEmail)
	case domain.AnomalyCacheInvalidatedEvent:
		return audit.HashEmail(ev.UserEmail)
	case domain.AnomalyCacheEvictedEvent:
		return audit.HashEmail(ev.UserEmail)
	case domain.RiskguardKillSwitchTrippedEvent:
		// Global kill-switch typically has empty UserEmail; hash falls
		// back to "" so the email_hash WHERE query excludes it (system
		// event, not user-correlated).
		if ev.UserEmail == "" {
			return ""
		}
		return audit.HashEmail(ev.UserEmail)
	case domain.RiskguardDailyCounterResetEvent:
		return audit.HashEmail(ev.UserEmail)
	case domain.RiskguardRejectionEvent:
		return audit.HashEmail(ev.UserEmail)
	case domain.TelegramSubscribedEvent:
		return audit.HashEmail(ev.UserEmail)
	case domain.TelegramChatBoundEvent:
		return audit.HashEmail(ev.UserEmail)
	case domain.OrderRejectedEvent:
		return audit.HashEmail(ev.Email)
	case domain.PositionConvertedEvent:
		return audit.HashEmail(ev.Email)
	case domain.PaperOrderRejectedEvent:
		return audit.HashEmail(ev.Email)
	case domain.MFOrderRejectedEvent:
		return audit.HashEmail(ev.Email)
	case domain.MFOrderPlacedEvent:
		return audit.HashEmail(ev.Email)
	case domain.MFOrderCancelledEvent:
		return audit.HashEmail(ev.Email)
	case domain.MFSIPPlacedEvent:
		return audit.HashEmail(ev.Email)
	case domain.MFSIPCancelledEvent:
		return audit.HashEmail(ev.Email)
	case domain.GTTRejectedEvent:
		return audit.HashEmail(ev.Email)
	case domain.GTTPlacedEvent:
		return audit.HashEmail(ev.Email)
	case domain.GTTModifiedEvent:
		return audit.HashEmail(ev.Email)
	case domain.GTTDeletedEvent:
		return audit.HashEmail(ev.Email)
	case domain.TrailingStopTriggeredEvent:
		return audit.HashEmail(ev.Email)
	case domain.TrailingStopSetEvent:
		return audit.HashEmail(ev.Email)
	case domain.TrailingStopCancelledEvent:
		return audit.HashEmail(ev.Email)
	case domain.NativeAlertPlacedEvent:
		return audit.HashEmail(ev.Email)
	case domain.NativeAlertModifiedEvent:
		return audit.HashEmail(ev.Email)
	case domain.NativeAlertDeletedEvent:
		return audit.HashEmail(ev.Email)
	case domain.PaperTradingEnabledEvent:
		return audit.HashEmail(ev.Email)
	case domain.PaperTradingDisabledEvent:
		return audit.HashEmail(ev.Email)
	case domain.PaperTradingResetEvent:
		return audit.HashEmail(ev.Email)
	case domain.GlobalFreezeEvent:
		// System event — no user-association field. Empty hash means
		// "this row is not user-correlated" (the email_hash WHERE
		// query won't include it, which is correct).
		return ""
	default:
		return ""
	}
}

// deriveAggregateID extracts the most meaningful aggregate identifier from a domain event.
func deriveAggregateID(e domain.Event) string {
	switch ev := e.(type) {
	case domain.OrderPlacedEvent:
		return ev.OrderID
	case domain.OrderModifiedEvent:
		return ev.OrderID
	case domain.OrderCancelledEvent:
		return ev.OrderID
	case domain.PositionOpenedEvent:
		return domain.PositionAggregateID(ev.Email, ev.Instrument, ev.Product)
	case domain.PositionClosedEvent:
		return domain.PositionAggregateID(ev.Email, ev.Instrument, ev.Product)
	case domain.AlertCreatedEvent:
		return ev.AlertID
	case domain.AlertTriggeredEvent:
		return ev.AlertID
	case domain.AlertDeletedEvent:
		return ev.AlertID
	case domain.UserFrozenEvent:
		return ev.Email
	case domain.UserSuspendedEvent:
		return ev.Email
	case domain.GlobalFreezeEvent:
		return ev.By
	case domain.FamilyInvitedEvent:
		return ev.AdminEmail
	case domain.FamilyMemberRemovedEvent:
		return ev.AdminEmail
	case domain.RiskLimitBreachedEvent:
		return ev.Email
	case domain.SessionCreatedEvent:
		return ev.SessionID
	case domain.TierChangedEvent:
		return ev.UserEmail
	case domain.WatchlistCreatedEvent:
		return ev.WatchlistID
	case domain.WatchlistDeletedEvent:
		return ev.WatchlistID
	case domain.WatchlistItemAddedEvent:
		return ev.WatchlistID
	case domain.WatchlistItemRemovedEvent:
		return ev.WatchlistID
	case domain.AnomalyBaselineSnapshottedEvent:
		return domain.AnomalyCacheAggregateID(ev.UserEmail)
	case domain.AnomalyCacheInvalidatedEvent:
		return domain.AnomalyCacheAggregateID(ev.UserEmail)
	case domain.AnomalyCacheEvictedEvent:
		return domain.AnomalyCacheAggregateID(ev.UserEmail)
	case domain.PluginRegisteredEvent:
		return domain.PluginWatcherAggregateID(ev.Path)
	case domain.PluginUnregisteredEvent:
		return domain.PluginWatcherAggregateID(ev.Path)
	case domain.PluginReloadTriggeredEvent:
		return domain.PluginWatcherAggregateID(ev.Path)
	case domain.PluginWatcherStartedEvent:
		return domain.PluginWatcherAggregateID("")
	case domain.PluginWatcherStoppedEvent:
		return domain.PluginWatcherAggregateID("")
	case domain.RiskguardKillSwitchTrippedEvent:
		return domain.RiskguardCountersAggregateID(ev.UserEmail)
	case domain.RiskguardDailyCounterResetEvent:
		return domain.RiskguardCountersAggregateID(ev.UserEmail)
	case domain.RiskguardRejectionEvent:
		return domain.RiskguardCountersAggregateID(ev.UserEmail)
	case domain.TelegramSubscribedEvent:
		return domain.TelegramSubscriptionAggregateID(ev.UserEmail)
	case domain.TelegramChatBoundEvent:
		return domain.TelegramSubscriptionAggregateID(ev.UserEmail)
	case domain.OrderRejectedEvent:
		// When OrderID is non-empty (modify/cancel rejections) the event
		// joins the existing order aggregate stream so a forensic walk
		// of the order ID sees place→reject inline. When OrderID is
		// empty (place_order failures, no broker ID issued) it falls
		// back to a per-rejection synthetic key built from email + the
		// event's own timestamp. See domain.OrderRejectedAggregateID.
		return domain.OrderRejectedAggregateID(ev.OrderID, ev.Email, ev.Timestamp)
	case domain.PositionConvertedEvent:
		// Keyed by (email, exchange, tradingsymbol, OLD product) so a
		// CNC->MIS->CNC sequence threads through a stable aggregate
		// stream rooted on the original holding's product. Matches the
		// pre-ES untyped key shape so existing rows aren't orphaned.
		return domain.PositionConvertedAggregateID(ev.Email, ev.Instrument.Exchange, ev.Instrument.Tradingsymbol, ev.OldProduct)
	case domain.PaperOrderRejectedEvent:
		// Paper IDs ("PAPER_<n>") are process-unique via atomic counter,
		// no email prefix needed. Empty OrderID (defence in depth) lands
		// in "paper:unknown" rather than colliding with real rows.
		return domain.PaperOrderAggregateID(ev.OrderID)
	case domain.MFOrderRejectedEvent:
		// Mirrors OrderRejectedAggregateID: non-empty OrderID joins
		// the existing MF aggregate stream; empty falls back to
		// synthetic "mf-rejected:<email>:<rfc3339-nanos>" key.
		return domain.MFOrderRejectedAggregateID(ev.OrderID, ev.Email, ev.Timestamp)
	case domain.GTTRejectedEvent:
		// Non-zero TriggerID joins the existing GTT aggregate stream
		// (matches the appendAuxEvent success path's "<id>" key shape);
		// zero falls back to synthetic "gtt-rejected:<email>:<ts>".
		return domain.GTTRejectedAggregateID(ev.TriggerID, ev.Email, ev.Timestamp)
	case domain.TrailingStopTriggeredEvent:
		// Keyed by TrailingStopID — uuid-derived 8-char prefix, globally
		// unique across users. The trailing stop's full lifecycle (set
		// -> N triggers -> cancel) replays under one aggregate stream.
		return domain.TrailingStopAggregateID(ev.TrailingStopID)
	case domain.MFOrderPlacedEvent:
		// Keyed by OrderID; pairs with MFOrderRejectedEvent (cancel
		// source) under the same aggregate when both fire.
		return domain.MFAggregateID(ev.OrderID)
	case domain.MFOrderCancelledEvent:
		return domain.MFAggregateID(ev.OrderID)
	case domain.MFSIPPlacedEvent:
		// Distinct ID namespace from MFOrder; same MFAggregateID helper
		// since SIPID is a Kite-assigned string just like OrderID.
		return domain.MFAggregateID(ev.SIPID)
	case domain.MFSIPCancelledEvent:
		return domain.MFAggregateID(ev.SIPID)
	case domain.GTTPlacedEvent:
		// fmt.Sprintf("%d", triggerID) matches the existing aux-event
		// aggregate-ID shape so existing rows and new typed events sort
		// under the same stream.
		return domain.GTTAggregateID(ev.TriggerID)
	case domain.GTTModifiedEvent:
		return domain.GTTAggregateID(ev.TriggerID)
	case domain.GTTDeletedEvent:
		return domain.GTTAggregateID(ev.TriggerID)
	case domain.TrailingStopSetEvent:
		// Same TrailingStopID-keyed routing as TrailingStopTriggeredEvent
		// so set->triggers->cancel replays under one aggregate stream.
		return domain.TrailingStopAggregateID(ev.TrailingStopID)
	case domain.TrailingStopCancelledEvent:
		return domain.TrailingStopAggregateID(ev.TrailingStopID)
	case domain.NativeAlertPlacedEvent:
		// UUID is empty at place time (broker assigns lazily); helper
		// falls back to email when UUID is empty (matching the prior
		// PlaceNativeAlertUseCase aggregate-id choice).
		return domain.NativeAlertAggregateID(ev.UUID, ev.Email)
	case domain.NativeAlertModifiedEvent:
		return domain.NativeAlertAggregateID(ev.UUID, ev.Email)
	case domain.NativeAlertDeletedEvent:
		return domain.NativeAlertAggregateID(ev.UUID, ev.Email)
	case domain.PaperTradingEnabledEvent:
		// Keyed by email — paper account is per-user, full enable/reset/
		// disable lifecycle replays under one stream.
		return domain.PaperTradingAggregateID(ev.Email)
	case domain.PaperTradingDisabledEvent:
		return domain.PaperTradingAggregateID(ev.Email)
	case domain.PaperTradingResetEvent:
		return domain.PaperTradingAggregateID(ev.Email)
	default:
		return "unknown"
	}
}
