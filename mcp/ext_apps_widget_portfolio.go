package mcp

import (
	"context"

	"github.com/algo2go/kite-mcp-audit"
	"github.com/algo2go/kite-mcp-cqrs"
)

// portfolioData fetches holdings + positions via the portfolio widget use case.
func portfolioData(ctx context.Context, manager extAppManagerPort, _ *audit.Store, email string) any {
	result, err := manager.QueryBus().DispatchWithResult(ctx, cqrs.GetPortfolioForWidgetQuery{Email: email})
	if err != nil {
		return map[string]string{"error": err.Error()}
	}
	return result
}
