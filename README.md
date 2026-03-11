# golinks

A local go-link server for macOS. Type `go/gh` in your browser and get redirected to GitHub (or wherever you configure).

## Features

- **Short links** -- `go/<phrase>` redirects to any URL you configure
- **Web UI** -- manage all your links at `go/` (add, edit, delete)
- **JSON API** -- `/_/api/links` for programmatic access
- **Auto-start** -- runs as a macOS LaunchDaemon, survives reboots
- **Zero dependencies** -- single Go binary, stdlib only

## Quick Start

```bash
# Clone and install
git clone https://github.com/mayks/golinks.git
cd golinks
./install.sh
```

The install script will:
1. Build the binary
2. Copy it to `/usr/local/bin/golinks`
3. Add `127.0.0.1 go` to `/etc/hosts`
4. Install and start a LaunchDaemon (auto-starts on boot)

Then open [http://go/](http://go/) in your browser.

## Usage

### Web UI

Visit `go/` to see all your links. From there you can:
- Add new links with the form at the top
- Edit any link inline (click Edit, change the URL, click Save)
- Delete links (click Delete)

### Browser

Just type `go/<phrase>` in your address bar:
- `go/gh` --> GitHub
- `go/mail` --> Gmail
- `go/cal` --> Google Calendar

### API

```bash
# Add a link
curl -X POST http://go/_/api/links \
  -H 'Content-Type: application/json' \
  -d '{"phrase":"gh","url":"https://github.com"}'

# Update a link
curl -X PUT http://go/_/api/links/gh \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/mayks"}'

# Delete a link
curl -X DELETE http://go/_/api/links/gh
```

## Configuration

Links are stored as JSON at `/usr/local/etc/golinks/links.json`. Override this path with the `GOLINKS_DATA` environment variable.

**Logs:**
- stdout: `/tmp/golinks.log`
- stderr: `/tmp/golinks.err`

## Uninstall

```bash
# Stop and remove the LaunchDaemon
sudo launchctl bootout system/com.golinks.server
sudo rm /Library/LaunchDaemons/com.golinks.server.plist

# Remove the binary
sudo rm /usr/local/bin/golinks

# Remove the hosts entry
sudo sed -i '' '/127.0.0.1 go/d' /etc/hosts

# Optionally remove link data
sudo rm -rf /usr/local/etc/golinks
```

## Development

```bash
# Run tests
go test -v

# Build
go build -o golinks .

# Run locally (requires sudo for port 80)
sudo ./golinks
```

## Cross-compile for Linux (e.g., Raspberry Pi)

```bash
GOOS=linux GOARCH=arm64 go build -o golinks-linux .
```
