package app

import (
	"github.com/algo2go/kite-mcp-bootstrap/kc"
	"github.com/algo2go/kite-mcp-alerts"
	"github.com/algo2go/kite-mcp-riskguard"
	tgbot "github.com/algo2go/kite-mcp-telegram"
)

// telegramManagerAdapter bridges *kc.Manager to telegram.KiteManager.
//
// Phase 3a kc/-side migration: the bot's KiteManager interface no longer
// returns concrete pointer types — it returns telegram-local narrow ports
// (AlertLookup, WatchlistLookup, InstrumentLookup, PaperEngineLookup,
// TickerLookup). Each accessor below returns the same concrete pointer
// the manager owns; Go's structural-interface satisfaction lets the
// concrete type pass through as the narrower port without an explicit
// cast. nil-safety: each method returns the typed nil so the call-site
// `if x == nil` guards in commands.go / trading_commands.go work
// correctly even when the concrete underlying type is unset (e.g.,
// PaperEngine disabled, TickerService not yet started).
type telegramManagerAdapter struct {
	m *kc.Manager
}

func (a *telegramManagerAdapter) TelegramStore() tgbot.TelegramLookup {
	return a.m.TelegramStore()
}
func (a *telegramManagerAdapter) AlertStore() tgbot.AlertLookup {
	if s := a.m.AlertStoreConcrete(); s != nil {
		return s
	}
	return nil
}
func (a *telegramManagerAdapter) WatchlistStore() tgbot.WatchlistLookup {
	if s := a.m.WatchlistStoreConcrete(); s != nil {
		return s
	}
	return nil
}
func (a *telegramManagerAdapter) GetAPIKeyForEmail(email string) string {
	return a.m.GetAPIKeyForEmail(email)
}
func (a *telegramManagerAdapter) GetAccessTokenForEmail(email string) string {
	return a.m.GetAccessTokenForEmail(email)
}
func (a *telegramManagerAdapter) TelegramNotifier() *alerts.TelegramNotifier {
	return a.m.TelegramNotifier()
}
func (a *telegramManagerAdapter) InstrumentsManager() tgbot.InstrumentLookup {
	if im := a.m.InstrumentsManagerConcrete(); im != nil {
		return im
	}
	return nil
}
func (a *telegramManagerAdapter) IsTokenValid(email string) bool {
	return a.m.IsTokenValid(email)
}
func (a *telegramManagerAdapter) RiskGuard() *riskguard.Guard {
	return a.m.RiskGuard()
}
func (a *telegramManagerAdapter) PaperEngine() tgbot.PaperEngineLookup {
	if pe := a.m.PaperEngineConcrete(); pe != nil {
		return pe
	}
	return nil
}
func (a *telegramManagerAdapter) TickerService() tgbot.TickerLookup {
	if ts := a.m.TickerServiceConcrete(); ts != nil {
		return ts
	}
	return nil
}
