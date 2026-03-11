# CLAUDE.md

## Project Overview

Local go-link server for macOS. Single Go binary that redirects `go/<phrase>` URLs to configured destinations, with a web UI for management.

## Tech Stack

- Go (stdlib only, no external dependencies)
- JSON file storage at `/usr/local/etc/golinks/links.json`
- Embedded HTML template via `//go:embed`

## Architecture

```
main.go        -- entry point, configures data path and starts HTTP server on :80
links.go       -- LinkStore: thread-safe CRUD backed by JSON file (atomic writes, rollback on failure)
handlers.go    -- HTTP handlers: redirect (302), index page, REST API (/_/api/links)
templates/
  index.html   -- embedded HTML template with inline CSS/JS
install.sh     -- build + install binary + /etc/hosts + LaunchDaemon setup
```

## Key Design Decisions

- Data path defaults to `/usr/local/etc/golinks/links.json` (not `$HOME`) because the LaunchDaemon runs as root without `$HOME` set. Override with `GOLINKS_DATA` env var.
- API routes use `/_/api/` prefix to avoid collisions with user-defined go-link phrases.
- File writes are atomic (write to .tmp then rename) with in-memory rollback on save failure.

## Commands

```bash
# Run all tests
go test -v

# Build
go build -o golinks .

# Install (builds, copies binary, sets up hosts + LaunchDaemon)
./install.sh
```

## Testing

15 tests total:
- 9 in `links_test.go` -- storage layer (CRUD, persistence, error cases)
- 6 in `handlers_test.go` -- HTTP handlers (redirect, API, index page)
