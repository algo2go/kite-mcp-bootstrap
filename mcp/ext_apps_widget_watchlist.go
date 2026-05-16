package mcp

import (
	"context"
	"math"
	"sort"

	"github.com/algo2go/kite-mcp-broker"
	"github.com/algo2go/kite-mcp-audit"
)

// watchlistData fetches all watchlists with items and LTP for the watchlist widget.
func watchlistData(_ context.Context, manager extAppManagerPort, _ *audit.Store, email string) any {
	store := manager.WatchlistStore()
	if store == nil {
		return nil
	}

	watchlists := store.ListWatchlists(email)
	if len(watchlists) == 0 {
		return map[string]any{"watchlists": []any{}, "total_count": 0}
	}

	// Sort by sort_order for consistent tab order.
	sort.Slice(watchlists, func(i, j int) bool {
		return watchlists[i].SortOrder < watchlists[j].SortOrder
	})

	// Collect all instruments across all watchlists for batch LTP.
	type itemWithLTP struct {
		Exchange         string  `json:"exchange"`
		Tradingsymbol    string  `json:"tradingsymbol"`
		Notes            string  `json:"notes,omitempty"`
		TargetEntry      float64 `json:"target_entry,omitempty"`
		TargetExit       float64 `json:"target_exit,omitempty"`
		LTP              float64 `json:"ltp,omitempty"`
		DistanceEntryPct float64 `json:"distance_entry_pct,omitempty"`
		DistanceExitPct  float64 `json:"distance_exit_pct,omitempty"`
		NearTarget       bool    `json:"near_target,omitempty"`
	}

	// Build per-watchlist item lists and collect instrument IDs.
	type wlEntry struct {
		ID    string        `json:"id"`
		Name  string        `json:"name"`
		Items []itemWithLTP `json:"items"`
	}

	entries := make([]wlEntry, 0, len(watchlists))
	var allInstruments []string
	instrumentSet := make(map[string]bool)

	for _, wl := range watchlists {
		items := store.GetItems(wl.ID)
		entry := wlEntry{ID: wl.ID, Name: wl.Name, Items: make([]itemWithLTP, 0, len(items))}
		for _, item := range items {
			entry.Items = append(entry.Items, itemWithLTP{
				Exchange:      item.Exchange,
				Tradingsymbol: item.Tradingsymbol,
				Notes:         item.Notes,
				TargetEntry:   item.TargetEntry,
				TargetExit:    item.TargetExit,
			})
			inst := item.Exchange + ":" + item.Tradingsymbol
			if !instrumentSet[inst] {
				instrumentSet[inst] = true
				allInstruments = append(allInstruments, inst)
			}
		}
		entries = append(entries, entry)
	}

	// Batch LTP fetch (max 50 per call, same pattern as alertsData).
	ltpMap := make(map[string]float64)
	client := brokerClientForEmail(manager, email)
	if client != nil && len(allInstruments) > 0 {
		const batchSize = 50
		for i := 0; i < len(allInstruments); i += batchSize {
			end := min(i+batchSize, len(allInstruments))
			batch := allInstruments[i:end]
			ltps, err := RetryBrokerCall(func() (map[string]broker.LTP, error) {
				return client.GetLTP(batch...)
			}, 2)
			if err == nil {
				for k, v := range ltps {
					ltpMap[k] = v.LastPrice
				}
			}
		}
	}

	// Enrich items with LTP and distance calculations.
	totalCount := 0
	for ei := range entries {
		for ii := range entries[ei].Items {
			item := &entries[ei].Items[ii]
			inst := item.Exchange + ":" + item.Tradingsymbol
			if ltp, ok := ltpMap[inst]; ok && ltp > 0 {
				item.LTP = ltp
				if item.TargetEntry > 0 {
					pct := ((ltp - item.TargetEntry) / item.TargetEntry) * 100
					item.DistanceEntryPct = pct
					if math.Abs(pct) <= 5.0 {
						item.NearTarget = true
					}
				}
				if item.TargetExit > 0 {
					pct := ((ltp - item.TargetExit) / item.TargetExit) * 100
					item.DistanceExitPct = pct
					if math.Abs(pct) <= 5.0 {
						item.NearTarget = true
					}
				}
			}
			totalCount++
		}
	}

	return map[string]any{
		"watchlists":  entries,
		"total_count": totalCount,
	}
}
