# GEAR - single command interface (casey/just)
# Docs site lives in ./docs (Docusaurus). Recipes below are the docs lifecycle.

set shell := ["bash", "-cu"]

# Alias: list all available recipes when run without arguments
default:
    @just --list

# Install docs dependencies
docs-install:
    cd docs && npm install

# Start the Docusaurus dev server (default http://localhost:3000)
docs-start:
    cd docs && npm start

# Build the static docs site into docs/build
docs-build:
    cd docs && npm run build

# Build then serve the static site locally (validate production output)
docs-serve:
    cd docs && npm run build && npm run serve

# Build and run the docs Playwright browser tests (mermaid overlay, etc.)
docs-test:
    cd docs && npm run build && npx playwright test

# Clear Docusaurus caches
docs-clear:
    cd docs && npm run clear

# ============================================================================
# App lifecycle (Story 1.1 scaffold)
# Pinned tool versions — go run <module>@<version> so no global CLI installs
# are required. Override DATABASE_URL via the environment if needed.
# ============================================================================

MIGRATE_VERSION := "v4.19.1"
SQLC_VERSION    := "v1.31.1"
GOLANGCI_VERSION := "v2.13.2"
DATABASE_URL    := env_var_or_default("DATABASE_URL", "postgres://gear:gear@localhost:5432/gear?sslmode=disable")
DB_CONTAINER    := env_var_or_default("DB_CONTAINER", "gear-db")

# Abort with an actionable message when podman/podman-compose are missing
podman-check:
    @command -v podman >/dev/null 2>&1 || { echo "G.E.A.R. requires podman (podman-compose >= 1.0.6) to run the dev stack. Please install podman and retry." >&2; exit 1; }
    @command -v podman-compose >/dev/null 2>&1 || { echo "G.E.A.R. requires podman-compose >= 1.0.6. Please install it and retry." >&2; exit 1; }
    @podman compose version >/dev/null 2>&1 || { echo "podman compose is not functional. Check the podman-compose plugin and retry." >&2; exit 1; }

# Bring the db container up and wait until it accepts connections
# The compose healthcheck allows ~50s (10 x 5s); give the wait loop the same
# budget so a cold first start (image pull + initdb) is not aborted early.
db-wait: podman-check
    podman compose up -d db
    @i=0; until podman exec {{DB_CONTAINER}} pg_isready -U gear -d gear >/dev/null 2>&1; do i=$((i+1)); [ $i -ge 60 ] && { echo "database not ready after 60s" >&2; exit 1; }; sleep 1; done

# Start PostgreSQL 18 and apply pending migrations (idempotent)
db-up: migrate-up
    podman compose ps

# Stop the db container (keeps the named volume)
db-down: podman-check
    podman compose down

alias db-stop := db-down
alias db-shutdown := db-down

# Apply pending forward migrations
migrate-up: db-wait
    go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@{{MIGRATE_VERSION}} -path ./migrations --database "{{DATABASE_URL}}" up

# Roll back all applied migrations (used to prove a clean rebuild)
migrate-down: db-wait
    go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@{{MIGRATE_VERSION}} -path ./migrations --database "{{DATABASE_URL}}" down -all

# Ensure web dependencies exist before running the SPA (fresh checkout)
web-deps:
    @test -d web/node_modules || npm --prefix web ci

# Run the full dev stack: DB + API + Vite SPA
dev: web-deps db-up
    npm --prefix web run dev & \
    vite_pid=$!; \
    sleep 2; \
    kill -0 "$vite_pid" 2>/dev/null || { echo "vite failed to start (see output above)" >&2; exit 1; }; \
    trap 'kill "$vite_pid" 2>/dev/null; kill $(jobs -p) 2>/dev/null' EXIT INT TERM; \
    go run ./cmd/server

# Build all Go packages
build:
    go build ./cmd/... ./internal/...

# Run all Go and web tests
test:
    go test ./cmd/... ./internal/...
    npm --prefix web run test

# Vet all Go packages
vet:
    go vet ./cmd/... ./internal/...

# Lint Go (via pinned golangci-lint) and web (via eslint)
lint: vet
    go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@{{GOLANGCI_VERSION}} run ./cmd/... ./internal/...
    npm --prefix web run lint

# Regenerate per-module stores from migrations/ via pinned sqlc (config: sqlc.yaml)
sqlc-generate:
    go run github.com/sqlc-dev/sqlc/cmd/sqlc@{{SQLC_VERSION}} generate
