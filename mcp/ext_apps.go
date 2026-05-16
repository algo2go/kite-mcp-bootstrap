package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/algo2go/kite-mcp-kc"
	"github.com/algo2go/kite-mcp-audit"
	"github.com/algo2go/kite-mcp-templates"
	"github.com/algo2go/kite-mcp-tools-common/common"
	"github.com/algo2go/kite-mcp-bootstrap/mcp/paper"
	"github.com/algo2go/kite-mcp-oauth"
)

// ResourceMIMEType is the MIME type that signals MCP App hosts (claude.ai,
// ChatGPT, VS Code) to render the resource as an interactive iframe widget.
const ResourceMIMEType = "text/html;profile=mcp-app"

// dataPlaceholder is replaced with pre-injected JSON in widget HTML.
const dataPlaceholder = `"__INJECTED_DATA__"`

// cssPlaceholder is replaced with the contents of dashboard-base.css at serve
// time so that widget HTML files don't need to duplicate the common variables
// and reset styles inline.
const cssPlaceholder = "/*__INJECTED_CSS__*/"

// extAppManagerPort is the narrow port surface ext_apps DataFuncs and the
// plugin_widget_* DataFuncs use. *kc.Manager satisfies this interface
// implicitly via its provider assertions in kc/manager_interfaces.go;
// 80+ test callsites that pass `mgr` to a DataFunc continue to compile
// unchanged because the manager's concrete type still matches.
//
// Phase 3a Batch 6b cleavage: previously each DataFunc signature took
// *kc.Manager directly, leaking the full manager surface into the
// widget data layer. Narrowing to a UNION of provider interfaces
// (a) prevents new DataFuncs from accidentally reaching for accessors
// outside the documented surface, (b) lets tests mock with a stub that
// implements just this interface, and (c) closes the last big "service
// locator" pocket flagged in Phase 3a Batch 5's deferral note.
type extAppManagerPort interface {
	// StoreAccessor composes the 15 store-provider interfaces (token,
	// credential, alert, telegram, watchlist, user, registry, audit,
	// billing, ticker, paper, instruments, alertDB, riskguard, mcpServer)
	// — gives the DataFuncs all the read-side store access they need
	// without listing each provider individually.
	kc.StoreAccessor
	// QueryBusProvider for CQRS read dispatch.
	kc.QueryBusProvider
	// AppConfigProvider for IsLocalMode + ExternalURL.
	kc.AppConfigProvider
	// BrokerResolverProvider for brokerClientForEmail's SessionSvc()
	// access in setupData/credentialsData.
	kc.BrokerResolverProvider
	// GetActiveSessionCount for the admin DataFuncs that report session
	// count (live in the same admin-only DataFunc closures that need
	// UserStore-based admin checks).
	GetActiveSessionCount() int
}

// appResource defines a UI resource served as an MCP App widget.
type appResource struct {
	URI          string
	Name         string
	TemplateFile string // *_app.html widget file
	// DataFunc returns JSON-serializable data to inject as window.__DATA__.
	// Receives the authenticated email. Returns nil if unauthenticated.
	DataFunc func(ctx context.Context, manager extAppManagerPort, auditStore *audit.Store, email string) any
}

// appResources lists the widget pages exposed as MCP App resources.
// Each uses a dedicated *_app.html optimized for iframe rendering
// (no external deps, AppBridge communication, pre-injected data).
//
// Per-widget DataFuncs (portfolioData, activityData, …) live in their
// own ext_apps_widget_<name>.go files within package mcp. The four
// inline admin closures below are kept inline because they are short,
// admin-only, and depend on each other's shape (active_sessions /
// total_users / global_freeze fields) — extracting them to standalone
// files would not improve readability the way per-widget extraction did
// for the user-facing 13.
var appResources = []appResource{
	{
		URI: "ui://kite-mcp/portfolio", Name: "Portfolio Widget",
		TemplateFile: "portfolio_app.html",
		DataFunc:     portfolioData,
	},
	{
		URI: "ui://kite-mcp/activity", Name: "Activity Widget",
		TemplateFile: "activity_app.html",
		DataFunc:     activityData,
	},
	{
		URI: "ui://kite-mcp/orders", Name: "Orders Widget",
		TemplateFile: "orders_app.html",
		DataFunc:     ordersData,
	},
	{
		URI: "ui://kite-mcp/alerts", Name: "Alerts Widget",
		TemplateFile: "alerts_app.html",
		DataFunc:     alertsData,
	},
	{
		URI: "ui://kite-mcp/paper", Name: "Paper Trading Widget",
		TemplateFile: "paper_app.html",
		DataFunc:     paperData,
	},
	{
		URI: "ui://kite-mcp/safety", Name: "Safety Widget",
		TemplateFile: "safety_app.html",
		DataFunc:     safetyData,
	},
	{
		URI: "ui://kite-mcp/order-form", Name: "Order Form Widget",
		TemplateFile: "order_form_app.html",
		DataFunc:     orderFormData,
	},
	{
		URI: "ui://kite-mcp/watchlist", Name: "Watchlist Widget",
		TemplateFile: "watchlist_app.html",
		DataFunc:     watchlistData,
	},
	{
		URI: "ui://kite-mcp/hub", Name: "Hub Widget",
		TemplateFile: "hub_app.html",
		DataFunc:     hubData,
	},
	{
		URI: "ui://kite-mcp/options-chain", Name: "Options Chain Widget",
		TemplateFile: "options_chain_app.html",
		DataFunc:     optionsChainData,
	},
	{
		URI: "ui://kite-mcp/chart", Name: "Chart Widget",
		TemplateFile: "chart_app.html",
		DataFunc:     chartData,
	},
	{
		URI: "ui://kite-mcp/setup", Name: "Setup Widget",
		TemplateFile: "setup_app.html",
		DataFunc:     setupData,
	},
	{
		URI: "ui://kite-mcp/credentials", Name: "Credentials Widget",
		TemplateFile: "credentials_app.html",
		DataFunc:     credentialsData,
	},
	{
		URI: "ui://kite-mcp/admin-overview", Name: "Admin Overview",
		TemplateFile: "admin_overview_app.html",
		DataFunc: func(_ context.Context, manager extAppManagerPort, auditStore *audit.Store, email string) any {
			if uStore := manager.UserStore(); uStore == nil || !uStore.IsAdmin(email) {
				return nil
			}
			resp := map[string]any{
				"active_sessions": manager.GetActiveSessionCount(),
				"uptime":          time.Since(common.ServerStartTime).Truncate(time.Second).String(),
				"go_version":      runtime.Version(),
				"goroutines":      runtime.NumGoroutine(),
			}
			if uStore := manager.UserStore(); uStore != nil {
				resp["total_users"] = uStore.Count()
			}
			if rg := manager.RiskGuard(); rg != nil {
				resp["global_freeze"] = rg.GetGlobalFreezeStatus()
			}
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			resp["heap_alloc_mb"] = float64(memStats.HeapAlloc) / 1024 / 1024
			if memStats.NumGC > 0 {
				resp["gc_pause_ms"] = float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1e6
			}
			return resp
		},
	},
	{
		URI: "ui://kite-mcp/admin-users", Name: "Admin Users",
		TemplateFile: "admin_users_app.html",
		DataFunc: func(_ context.Context, manager extAppManagerPort, auditStore *audit.Store, email string) any {
			if uStore := manager.UserStore(); uStore == nil || !uStore.IsAdmin(email) {
				return nil
			}
			users := manager.UserStore().List()
			type entry struct {
				Email       string `json:"email"`
				Role        string `json:"role"`
				Status      string `json:"status"`
				CreatedAt   string `json:"created_at"`
				LastLogin   string `json:"last_login,omitempty"`
				OnboardedBy string `json:"onboarded_by"`
			}
			entries := make([]entry, 0, len(users))
			for _, u := range users {
				var ll string
				if !u.LastLogin.IsZero() {
					ll = u.LastLogin.Format(time.RFC3339)
				}
				entries = append(entries, entry{
					Email: u.Email, Role: u.Role, Status: u.Status,
					CreatedAt: u.CreatedAt.Format(time.RFC3339), LastLogin: ll, OnboardedBy: u.OnboardedBy,
				})
			}
			return map[string]any{"total": len(users), "from": 0, "limit": len(users), "users": entries}
		},
	},
	{
		URI: "ui://kite-mcp/admin-metrics", Name: "Admin Metrics",
		TemplateFile: "admin_metrics_app.html",
		DataFunc: func(_ context.Context, manager extAppManagerPort, auditStore *audit.Store, email string) any {
			if uStore := manager.UserStore(); uStore == nil || !uStore.IsAdmin(email) {
				return nil
			}
			resp := map[string]any{"period": "24h", "active_sessions": manager.GetActiveSessionCount()}
			if auditStore != nil {
				since := time.Now().Add(-24 * time.Hour)
				if stats, err := auditStore.GetGlobalStats(since); err == nil {
					resp["total_calls"] = stats.TotalCalls
					resp["error_count"] = stats.ErrorCount
					resp["avg_latency_ms"] = stats.AvgLatencyMs
					if stats.TotalCalls > 0 {
						resp["error_rate"] = fmt.Sprintf("%.1f%%", float64(stats.ErrorCount)/float64(stats.TotalCalls)*100)
					} else {
						resp["error_rate"] = "0.0%"
					}
				}
				if metrics, err := auditStore.GetToolMetrics(since); err == nil {
					resp["tool_metrics"] = metrics
				}
			}
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			resp["heap_alloc_mb"] = float64(memStats.HeapAlloc) / 1024 / 1024
			resp["goroutines"] = runtime.NumGoroutine()
			return resp
		},
	},
	{
		URI: "ui://kite-mcp/admin-registry", Name: "Admin Registry",
		TemplateFile: "admin_registry_app.html",
		DataFunc: func(_ context.Context, manager extAppManagerPort, auditStore *audit.Store, email string) any {
			if uStore := manager.UserStore(); uStore == nil || !uStore.IsAdmin(email) {
				return nil
			}
			// Phase 3a kc/-side migration (Hex 99→100 close-out): route
			// through the RegistryStore() port (RegistryReader.List
			// satisfies the only call below) instead of the prior
			// RegistryStoreConcrete() leak. Symmetric with the
			// admin_baseline / admin_cache_info forensics-only escape
			// hatches: those NEED concrete because UserOrderStats /
			// StatsCacheHitRate are not on AuditStoreInterface — the
			// list-all-registrations path does NOT need concrete.
			regStore := manager.RegistryStore()
			if regStore == nil {
				return map[string]any{"total": 0, "active": 0, "registrations": []any{}}
			}
			regs := regStore.List()
			active := 0
			for _, r := range regs {
				if r.Status == "active" {
					active++
				}
			}
			return map[string]any{"total": len(regs), "active": active, "registrations": regs}
		},
	},
}

// pagePathToResourceURI maps dashboard URL paths to ui:// resource URIs.
// #nosec G101 -- values are ui:// MCP resource URI constants (widget
// identifiers per the MCP Apps spec), not credentials. gosec's regex
// trips on the URI scheme pattern but there is nothing secret here.
var pagePathToResourceURI = map[string]string{
	"/dashboard":             "ui://kite-mcp/portfolio",
	"/dashboard/activity":    "ui://kite-mcp/activity",
	"/dashboard/orders":      "ui://kite-mcp/orders",
	"/dashboard/alerts":      "ui://kite-mcp/alerts",
	"/dashboard/paper":       "ui://kite-mcp/paper",
	"/dashboard/safety":      "ui://kite-mcp/safety",
	"/dashboard/order-form":  "ui://kite-mcp/order-form",
	"/dashboard/watchlist":   "ui://kite-mcp/watchlist",
	"/dashboard/hub":         "ui://kite-mcp/hub",
	"/dashboard/options":     "ui://kite-mcp/options-chain",
	"/dashboard/chart":       "ui://kite-mcp/chart",
	"/dashboard/setup":       "ui://kite-mcp/setup",
	"/dashboard/credentials": "ui://kite-mcp/credentials",
	"/admin/overview":        "ui://kite-mcp/admin-overview",
	"/admin/users":           "ui://kite-mcp/admin-users",
	"/admin/metrics":         "ui://kite-mcp/admin-metrics",
	"/admin/registry":        "ui://kite-mcp/admin-registry",
}

// withAppUI sets the flat _meta["ui/resourceUri"] key on a tool definition.
// Claude.ai only recognizes this flat format (not nested _meta.ui.resourceUri).
// The ext-apps SDK's getToolUiResourceUri() accepts both formats.
//
// The OpenAI Apps SDK reads the same resource URI from `_meta["openai/outputTemplate"]`
// (see developers.openai.com/apps-sdk/build/mcp-server), so we set both keys to
// the same value — unlocking ChatGPT widget rendering with zero additional hosting.
// Both keys reference the same ui:// MCP resource served with MIME `text/html;profile=mcp-app`.
func withAppUI(t gomcp.Tool, resourceURI string) gomcp.Tool {
	if resourceURI == "" {
		return t
	}
	t.Meta = &gomcp.Meta{
		AdditionalFields: map[string]any{
			uiMetaKey:     resourceURI,
			openAIMetaKey: resourceURI,
		},
	}
	return t
}

// resourceURIForTool returns the ui:// resource URI for a tool based on its
// dashboard page mapping, or empty string if the tool has no associated page.
func resourceURIForTool(toolName string) string {
	pagePath, ok := paper.ToolDashboardPage[toolName]
	if !ok {
		return ""
	}
	return pagePathToResourceURI[pagePath]
}

// injectData replaces the dataPlaceholder in HTML with the JSON-encoded data.
// If data is nil, injects "null". Escapes "</script>" sequences in the JSON
// to prevent XSS breakout from the <script> tag context.
func injectData(html string, data any) string {
	var jsonStr string
	if data == nil {
		jsonStr = "null"
	} else {
		b, err := json.Marshal(data)
		if err != nil {
			jsonStr = "null"
		} else {
			jsonStr = string(b)
		}
	}
	// Defense-in-depth: Go's json.Marshal already escapes "<" as "<",
	// so these replacements are no-ops in practice. They guard against future
	// changes to the JSON encoding (e.g., SetEscapeHTML(false)).
	jsonStr = strings.ReplaceAll(jsonStr, "</", `<\/`)
	jsonStr = strings.ReplaceAll(jsonStr, "<!--", `<\!--`)
	// U+2028 (LINE SEPARATOR) and U+2029 (PARAGRAPH SEPARATOR) are valid
	// whitespace in JSON but illegal line terminators inside JS string
	// literals. Go's json.Marshal does NOT escape them, so if attacker-
	// controlled data contains one, the injected JSON literal terminates
	// early in the browser and subsequent bytes execute as script — XSS.
	// Escape to their \uXXXX form so they live harmlessly inside the string.
	jsonStr = strings.ReplaceAll(jsonStr, "\u2028", `\u2028`)
	jsonStr = strings.ReplaceAll(jsonStr, "\u2029", `\u2029`)
	return strings.Replace(html, dataPlaceholder, jsonStr, 1)
}

// RegisterAppResources registers widget HTML pages as MCP App resources.
// Each resource handler dynamically injects user-specific data so widgets
// render instantly without needing AppBridge round-trips for initial load.
//
// It also installs an OnAfterListTools hook that strips the
// `_meta["ui/resourceUri"]` key from every tool in the list_tools response
// when the calling client does NOT advertise the MCP Apps UI extension.
// This keeps non-widget hosts (Claude Code, Windsurf, Cursor pre-2.6, Zed,
// 5ire, Cline) free of noisy fallback text while leaving widget-capable
// hosts (Claude.ai, Claude Desktop, ChatGPT, VS Code Copilot, Goose)
// untouched. Widget resource registration is NOT gated — clients that
// don't understand ui:// simply won't request them.
func RegisterAppResources(srv *server.MCPServer, manager *kc.Manager, auditStore *audit.Store, logger *slog.Logger) {
	// Install per-session UI capability gating on tool listings. We append
	// via GetHooks so this wiring lives with the other ext-apps code and
	// doesn't require touching the top-level server construction. Multiple
	// calls append additional hooks rather than replacing — mcp-go runs them
	// in registration order.
	if hooks := srv.GetHooks(); hooks != nil {
		hooks.AddAfterListTools(func(ctx context.Context, _ any, _ *gomcp.ListToolsRequest, result *gomcp.ListToolsResult) {
			if result == nil {
				return
			}
			var caps gomcp.ClientCapabilities
			if session := server.ClientSessionFromContext(ctx); session != nil {
				if ci, ok := session.(server.SessionWithClientInfo); ok {
					caps = ci.GetClientCapabilities()
				}
			}
			if clientSupportsUI(caps) {
				return
			}
			result.Tools = stripUIResourceURIFromTools(result.Tools)
		})
	}

	// Read the shared base CSS once so it can be injected into every widget
	// that contains the cssPlaceholder token.
	cssBytes, _ := templates.FS.ReadFile("dashboard-base.css")
	cssContent := string(cssBytes)

	registered := 0
	for _, res := range appResources {
		htmlBytes, err := templates.FS.ReadFile(res.TemplateFile)
		if err != nil {
			logger.Warn("Failed to read widget template",
				"uri", res.URI, "file", res.TemplateFile, "error", err)
			continue
		}
		htmlTemplate := string(htmlBytes)
		if cssContent != "" {
			htmlTemplate = strings.Replace(htmlTemplate, cssPlaceholder, cssContent, 1)
		}

		srv.AddResource(
			gomcp.Resource{
				URI:      res.URI,
				Name:     res.Name,
				MIMEType: ResourceMIMEType,
			},
			func(ctx context.Context, req gomcp.ReadResourceRequest) ([]gomcp.ResourceContents, error) {
				// Extract authenticated email from MCP session context.
				email := oauth.EmailFromContext(ctx)
				var data any
				if email != "" && res.DataFunc != nil {
					data = res.DataFunc(ctx, manager, auditStore, email)
				}

				html := injectData(htmlTemplate, data)

				return []gomcp.ResourceContents{
					gomcp.TextResourceContents{
						URI:      res.URI,
						MIMEType: ResourceMIMEType,
						Text:     html,
					},
				}, nil
			},
		)
		registered++
	}

	// Also install every plugin-registered widget (see plugin_widgets.go).
	// Plugins supply their own ResourceContents — we don't template or
	// inject data for them; they're responsible for their own HTML
	// generation. The only safety net is that RegisterWidget rejects
	// URIs that collide with built-ins, so a plugin cannot hijack the
	// portfolio/activity/orders widgets.
	pluginWidgets := ListPluginWidgets()
	for _, pw := range pluginWidgets {
		pw := pw // capture
		srv.AddResource(
			gomcp.Resource{
				URI:      pw.URI,
				Name:     pw.Name,
				MIMEType: ResourceMIMEType,
			},
			func(ctx context.Context, req gomcp.ReadResourceRequest) ([]gomcp.ResourceContents, error) {
				// SafeCall wraps the plugin handler with panic
				// recovery; a buggy widget renders as an error
				// payload rather than taking down the MCP
				// ReadResource surface.
				contents, err := SafeCall(pw.URI, func() ([]gomcp.ResourceContents, error) {
					return pw.Handler(ctx, req)
				})
				if err != nil {
					ReportPluginHealth(pw.URI, HealthStatus{
						State:   HealthStateFailed,
						Message: "widget handler panic: " + err.Error(),
					})
				}
				return contents, err
			},
		)
	}
	if len(pluginWidgets) > 0 {
		logger.Info("Plugin widget resources registered", "count", len(pluginWidgets))
	}

	logger.Info("MCP App widget resources registered",
		"builtin", registered, "plugin", len(pluginWidgets))
}
