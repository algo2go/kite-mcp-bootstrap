package mcp

import (
	"os"

	gomcp "github.com/mark3labs/mcp-go/mcp"
)

// UICapabilityExtensionKey is the MCP Apps capability key that clients
// advertise in `initialize.capabilities.extensions` when they can render
// `ui://` resources as inline widgets (Claude.ai web, Claude Desktop,
// ChatGPT, VS Code Copilot, Goose). Hosts that do NOT advertise this key
// (Claude Code, Windsurf, Cursor pre-2.6, Zed, 5ire, Cline) receive noisy
// fallback text when the server returns widget resource references — so
// we strip `_meta["ui/resourceUri"]` from tool listings for those clients.
//
// Protocol reference: MCP 2026-01-26 spec, section on client extensions.
const UICapabilityExtensionKey = "io.modelcontextprotocol/ui"

// uiMetaKey is the flat `_meta` key that points a tool response at a
// ui:// resource. Kept in sync with withAppUI's key below.
const (
	uiMetaKey     = "ui/resourceUri"
	openAIMetaKey = "openai/outputTemplate"
)

// clientSupportsUI reports whether the given client capabilities
// advertise MCP Apps widget support.
//
// Detection order:
//  1. Operator kill-switch: MCP_UI_ENABLED=false disables widgets for
//     every client regardless of advertisement.
//  2. Capability advertisement: returns true if `extensions` or
//     `experimental` contains the MCP Apps extension key.
//
// Fail-closed on missing advertisement: clients that do NOT advertise
// the extension get widgets stripped. This is intentional — per the
// research, unsupported hosts (Claude Code, Windsurf, Cursor pre-2.6,
// Zed, 5ire, Cline) emit noisy fallback text otherwise. The env var
// remains as a forced-disable kill switch (e.g., if a widget release
// is buggy in production), but cannot force-enable on a client that
// didn't advertise — that would reintroduce the noise bug.
func clientSupportsUI(caps gomcp.ClientCapabilities) bool {
	return clientSupportsUIWithOverride(caps, os.Getenv("MCP_UI_ENABLED"))
}

// clientSupportsUIWithOverride is the pure capability-check used by tests.
// It takes the kill-switch string (typically MCP_UI_ENABLED) explicitly so
// table-driven tests can run in parallel without t.Setenv.
func clientSupportsUIWithOverride(caps gomcp.ClientCapabilities, killSwitch string) bool {
	if killSwitch == "false" {
		return false
	}
	if caps.Extensions != nil {
		if _, ok := caps.Extensions[UICapabilityExtensionKey]; ok {
			return true
		}
	}
	if caps.Experimental != nil {
		if _, ok := caps.Experimental[UICapabilityExtensionKey]; ok {
			return true
		}
	}
	return false
}

// stripUIResourceURIFromTools returns a copy of the given tools with the
// `ui/resourceUri` key removed from each tool's `_meta` block. Tools
// without _meta, or whose _meta has no ui/resourceUri key, are copied
// unchanged. The original tools are NOT mutated — we allocate fresh
// `_meta` maps so concurrent list_tools calls from UI-capable sessions
// still see the full metadata.
//
// Used by the OnAfterListTools hook installed in RegisterAppResources
// to quiet non-widget clients (Claude Code, Windsurf, Cursor pre-2.6,
// Zed, 5ire, Cline).
func stripUIResourceURIFromTools(tools []gomcp.Tool) []gomcp.Tool {
	if tools == nil {
		return nil
	}
	out := make([]gomcp.Tool, len(tools))
	for i, t := range tools {
		if t.Meta != nil && t.Meta.AdditionalFields != nil {
			_, hasUI := t.Meta.AdditionalFields[uiMetaKey]
			_, hasOpenAI := t.Meta.AdditionalFields[openAIMetaKey]
			if hasUI || hasOpenAI {
				newFields := make(map[string]any, len(t.Meta.AdditionalFields))
				for k, v := range t.Meta.AdditionalFields {
					if k == uiMetaKey || k == openAIMetaKey {
						continue
					}
					newFields[k] = v
				}
				t.Meta = &gomcp.Meta{
					ProgressToken:    t.Meta.ProgressToken,
					AdditionalFields: newFields,
				}
			}
		}
		out[i] = t
	}
	return out
}
