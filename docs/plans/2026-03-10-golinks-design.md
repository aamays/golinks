# Go-Links Server Design

## Overview

A local go-link server for macOS that redirects `go/<phrase>` URLs to configured destinations, with a web-based management UI.

## Decisions

- **Language:** Go — single binary, no dependencies
- **Storage:** JSON file at `~/.config/golinks/links.json`
- **DNS:** `/etc/hosts` entry (`127.0.0.1 go`), server on port 80
- **Auto-start:** macOS LaunchDaemon (`/Library/LaunchDaemons/com.golinks.server.plist`)
- **UI:** Server-rendered HTML with embedded templates (Go `embed` package)

## Architecture

Single Go binary with three responsibilities:

1. **Redirect handler** — `go/<phrase>` looks up phrase in JSON, returns 302 redirect
2. **Management UI** — `go/` serves index page listing all links with add/edit/delete
3. **API endpoints** — REST API the UI calls to manage links

```
Browser → go/<phrase> → Go server (port 80) → 302 redirect to target URL
Browser → go/          → Go server (port 80) → index page
```

## Data Model

Simple JSON object mapping phrases to URLs:

```json
{
  "gh": "https://github.com",
  "mail": "https://gmail.com",
  "cal": "https://calendar.google.com"
}
```

## API Endpoints

| Method | Path | Action |
|--------|------|--------|
| GET | `/{phrase}` | Redirect to target URL (302) |
| GET | `/` | Serve index page with all links |
| POST | `/_/api/links` | Add a new link `{phrase, url}` |
| PUT | `/_/api/links/{phrase}` | Update a link's URL |
| DELETE | `/_/api/links/{phrase}` | Delete a link |

`/_/api/` prefix avoids collisions with user-defined phrases.

## Index Page

Server-rendered HTML with:
- Table showing all links (phrase, URL) with edit/delete buttons
- Form to add new links
- Inline editing
- Clean, minimal CSS (no framework)

## Deployment

- `/etc/hosts`: `127.0.0.1 go`
- Binary: `/usr/local/bin/golinks`
- LaunchDaemon: `/Library/LaunchDaemons/com.golinks.server.plist`

## File Structure

```
golinks/
├── main.go              # Entry point, server setup, routing
├── links.go             # Link storage (read/write JSON)
├── handlers.go          # HTTP handlers (redirect, API, index)
├── templates/
│   └── index.html       # Embedded HTML template
├── go.mod
└── docs/plans/
    └── 2026-03-10-golinks-design.md
```
