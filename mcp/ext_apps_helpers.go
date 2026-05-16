package mcp

import (
	"github.com/algo2go/kite-mcp-broker"
)

// brokerClientForEmail resolves a broker.Client for the given email,
// or nil if credentials/token are not available.
//
// Phase 3a Batch 6b: signature narrowed to extAppManagerPort
// (which composes BrokerResolverProvider). *kc.Manager satisfies the
// interface so existing test callers compile unchanged.
func brokerClientForEmail(manager extAppManagerPort, email string) broker.Client {
	client, err := manager.GetBrokerForEmail(email)
	if err != nil {
		return nil
	}
	return client
}
