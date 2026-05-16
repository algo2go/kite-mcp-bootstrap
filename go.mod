module github.com/algo2go/kite-mcp-bootstrap

go 1.25.0

// Composition root for kite-mcp-server. Relocated 2026-05-16 from
// github.com/zerodha/kite-mcp-server in-tree packages.
//
// In-tree workspace members (resolved via go.work + replace directives):
//   - ./app/providers : Fx provider/recipe sub-module
//   - ./plugins       : plugin scaffolding sub-module
//   - ./testutil      : test fakes + fixtures sub-module
//
// External algo2go domain modules (28 total, fetched from GOPROXY):
//   alerts, aop, audit, billing, broker, clockport, cqrs, decorators,
//   domain, eventsourcing, i18n, instruments, isttz, legaldocs, logger,
//   money, oauth, papertrading, registry, riskguard, scheduler, sectors,
//   telegram, templates, ticker, usecases, users, watchlist.
//
// Versions match kite-mcp-server's go.mod at HEAD `b6b4f6a` (the source
// of this relocation). Bump independently from here on.

require (
	github.com/algo2go/kite-mcp-alerts v0.6.0
	github.com/algo2go/kite-mcp-audit v0.2.0
	github.com/algo2go/kite-mcp-billing v0.3.0
	github.com/algo2go/kite-mcp-bootstrap/app/providers v0.1.1
	github.com/algo2go/kite-mcp-bootstrap/plugins v0.1.1
	github.com/algo2go/kite-mcp-bootstrap/testutil v0.1.1
	github.com/algo2go/kite-mcp-broker v0.1.0
	github.com/algo2go/kite-mcp-clockport v0.1.0
	github.com/algo2go/kite-mcp-cqrs v0.1.0
	github.com/algo2go/kite-mcp-decorators v0.1.0
	github.com/algo2go/kite-mcp-domain v0.1.0
	github.com/algo2go/kite-mcp-eventsourcing v0.1.0
	github.com/algo2go/kite-mcp-i18n v0.1.0
	github.com/algo2go/kite-mcp-instruments v0.1.0
	github.com/algo2go/kite-mcp-isttz v0.1.0
	github.com/algo2go/kite-mcp-kc v0.1.0
	github.com/algo2go/kite-mcp-legaldocs v0.1.0
	github.com/algo2go/kite-mcp-logger v0.1.0
	github.com/algo2go/kite-mcp-metrics v0.1.0
	github.com/algo2go/kite-mcp-money v0.1.0
	github.com/algo2go/kite-mcp-oauth v0.1.0
	github.com/algo2go/kite-mcp-papertrading v0.1.0
	github.com/algo2go/kite-mcp-registry v0.1.0
	github.com/algo2go/kite-mcp-riskguard v0.1.0
	github.com/algo2go/kite-mcp-scheduler v0.1.0
	github.com/algo2go/kite-mcp-sectors v0.1.0
	github.com/algo2go/kite-mcp-telegram v0.1.0
	github.com/algo2go/kite-mcp-templates v0.1.0
	github.com/algo2go/kite-mcp-ticker v0.1.0
	github.com/algo2go/kite-mcp-usecases v0.1.0
	github.com/algo2go/kite-mcp-users v0.2.0
	github.com/algo2go/kite-mcp-watchlist v0.2.0
	github.com/fsnotify/fsnotify v1.9.0
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/google/uuid v1.6.0
	github.com/mark3labs/mcp-go v0.46.0
	github.com/stretchr/testify v1.11.1
	github.com/stripe/stripe-go/v82 v82.5.1
	github.com/yuin/goldmark v1.8.2
	github.com/zerodha/gokiteconnect/v4 v4.4.0
	go.uber.org/fx v1.24.0
	go.uber.org/goleak v1.3.0
	golang.org/x/crypto v0.48.0
	golang.org/x/time v0.15.0
	pgregory.net/rapid v1.2.0
)

// Workspace-member replace directives — point at the in-tree sub-modules so
// the workspace `go build` works without needing published tags for them.
replace (
	github.com/algo2go/kite-mcp-bootstrap/app/providers => ./app/providers
	github.com/algo2go/kite-mcp-bootstrap/plugins => ./plugins
	github.com/algo2go/kite-mcp-bootstrap/testutil => ./testutil
)

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/coder/websocket v1.8.12 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/gocarina/gocsv v0.0.0-20180809181117-b8c38cb1ba36 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-plugin v1.7.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.9.2 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/tursodatabase/libsql-client-go v0.0.0-20251219100830-236aa1ff8acc // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.67.6 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.46.1 // indirect
)
