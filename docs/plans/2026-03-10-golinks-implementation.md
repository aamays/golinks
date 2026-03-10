# Go-Links Server Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a local go-link server that redirects `go/<phrase>` URLs and provides a web UI for managing links.

**Architecture:** Single Go binary using stdlib `net/http`. JSON file storage at `~/.config/golinks/links.json`. Embedded HTML template via `embed` package. Runs on port 80, auto-started via macOS LaunchDaemon.

**Tech Stack:** Go (stdlib only, no external dependencies)

---

### Task 0: Install Go

**Step 1: Install Go via Homebrew**

Run: `brew install go`
Expected: Go installed successfully

**Step 2: Verify installation**

Run: `go version`
Expected: `go version go1.2x.x darwin/arm64` (or similar)

**Step 3: Initialize the Go module**

Run: `cd /Users/mayks/Development/golinks && go mod init github.com/mayks/golinks`
Expected: `go.mod` file created

**Step 4: Commit**

```bash
git add go.mod
git commit -m "chore: initialize Go module"
```

---

### Task 1: Link Storage Layer

**Files:**
- Create: `links.go`
- Create: `links_test.go`

**Step 1: Write the failing test for link storage**

```go
// links_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLinkStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	store, err := NewLinkStore(path)
	if err != nil {
		t.Fatalf("NewLinkStore: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestAddAndGet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	store, _ := NewLinkStore(path)

	err := store.Add("gh", "https://github.com")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	url, ok := store.Get("gh")
	if !ok {
		t.Fatal("expected to find 'gh'")
	}
	if url != "https://github.com" {
		t.Fatalf("got %q, want %q", url, "https://github.com")
	}
}

func TestAddDuplicate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	store, _ := NewLinkStore(path)

	store.Add("gh", "https://github.com")
	err := store.Add("gh", "https://github.com")
	if err == nil {
		t.Fatal("expected error for duplicate phrase")
	}
}

func TestUpdate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	store, _ := NewLinkStore(path)

	store.Add("gh", "https://github.com")
	err := store.Update("gh", "https://github.com/mayks")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	url, _ := store.Get("gh")
	if url != "https://github.com/mayks" {
		t.Fatalf("got %q after update", url)
	}
}

func TestUpdateNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	store, _ := NewLinkStore(path)

	err := store.Update("nope", "https://example.com")
	if err == nil {
		t.Fatal("expected error for missing phrase")
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	store, _ := NewLinkStore(path)

	store.Add("gh", "https://github.com")
	err := store.Delete("gh")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, ok := store.Get("gh")
	if ok {
		t.Fatal("expected 'gh' to be deleted")
	}
}

func TestAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	store, _ := NewLinkStore(path)

	store.Add("gh", "https://github.com")
	store.Add("mail", "https://gmail.com")

	all := store.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 links, got %d", len(all))
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	store, _ := NewLinkStore(path)

	store.Add("gh", "https://github.com")

	// Create new store from same file
	store2, err := NewLinkStore(path)
	if err != nil {
		t.Fatalf("NewLinkStore (reload): %v", err)
	}

	url, ok := store2.Get("gh")
	if !ok || url != "https://github.com" {
		t.Fatalf("persistence failed: ok=%v url=%q", ok, url)
	}
}

func TestCreatesDirIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "links.json")
	_, err := NewLinkStore(path)
	if err != nil {
		t.Fatalf("expected store to create parent dirs: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		t.Fatal("expected parent directories to be created")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestNewLinkStore`
Expected: FAIL — `NewLinkStore` not defined

**Step 3: Write minimal implementation**

```go
// links.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Link represents a single go-link entry.
type Link struct {
	Phrase string `json:"phrase"`
	URL    string `json:"url"`
}

// LinkStore manages go-links backed by a JSON file.
type LinkStore struct {
	mu    sync.RWMutex
	path  string
	links map[string]string // phrase -> URL
}

// NewLinkStore creates or loads a link store from the given JSON file path.
func NewLinkStore(path string) (*LinkStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	s := &LinkStore{
		path:  path,
		links: make(map[string]string),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read links file: %w", err)
	}

	if err := json.Unmarshal(data, &s.links); err != nil {
		return nil, fmt.Errorf("parse links file: %w", err)
	}

	return s, nil
}

// Get returns the URL for a phrase.
func (s *LinkStore) Get(phrase string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.links[phrase]
	return url, ok
}

// All returns all links sorted by phrase.
func (s *LinkStore) All() []Link {
	s.mu.RLock()
	defer s.mu.RUnlock()

	links := make([]Link, 0, len(s.links))
	for phrase, url := range s.links {
		links = append(links, Link{Phrase: phrase, URL: url})
	}
	sort.Slice(links, func(i, j int) bool {
		return links[i].Phrase < links[j].Phrase
	})
	return links
}

// Add creates a new link. Returns error if phrase already exists.
func (s *LinkStore) Add(phrase, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.links[phrase]; exists {
		return fmt.Errorf("phrase %q already exists", phrase)
	}

	s.links[phrase] = url
	return s.save()
}

// Update modifies an existing link. Returns error if phrase not found.
func (s *LinkStore) Update(phrase, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.links[phrase]; !exists {
		return fmt.Errorf("phrase %q not found", phrase)
	}

	s.links[phrase] = url
	return s.save()
}

// Delete removes a link. Returns error if phrase not found.
func (s *LinkStore) Delete(phrase string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.links[phrase]; !exists {
		return fmt.Errorf("phrase %q not found", phrase)
	}

	delete(s.links, phrase)
	return s.save()
}

func (s *LinkStore) save() error {
	data, err := json.MarshalIndent(s.links, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal links: %w", err)
	}
	return os.WriteFile(s.path, data, 0644)
}
```

**Step 4: Run all tests to verify they pass**

Run: `go test -v`
Expected: All 8 tests PASS

**Step 5: Commit**

```bash
git add links.go links_test.go
git commit -m "feat: add link storage layer with JSON persistence"
```

---

### Task 2: HTTP Handlers

**Files:**
- Create: `handlers.go`
- Create: `handlers_test.go`

**Step 1: Write the failing tests for handlers**

```go
// handlers_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func testStore(t *testing.T) *LinkStore {
	t.Helper()
	dir := t.TempDir()
	store, err := NewLinkStore(filepath.Join(dir, "links.json"))
	if err != nil {
		t.Fatalf("NewLinkStore: %v", err)
	}
	return store
}

func TestHandleRedirect(t *testing.T) {
	store := testStore(t)
	store.Add("gh", "https://github.com")

	handler := NewServer(store)

	req := httptest.NewRequest("GET", "/gh", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://github.com" {
		t.Fatalf("expected redirect to https://github.com, got %q", loc)
	}
}

func TestHandleRedirectNotFound(t *testing.T) {
	store := testStore(t)
	handler := NewServer(store)

	req := httptest.NewRequest("GET", "/nope", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleAPIAdd(t *testing.T) {
	store := testStore(t)
	handler := NewServer(store)

	body := strings.NewReader(`{"phrase":"gh","url":"https://github.com"}`)
	req := httptest.NewRequest("POST", "/_/api/links", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	url, ok := store.Get("gh")
	if !ok || url != "https://github.com" {
		t.Fatalf("link not stored: ok=%v url=%q", ok, url)
	}
}

func TestHandleAPIUpdate(t *testing.T) {
	store := testStore(t)
	store.Add("gh", "https://github.com")
	handler := NewServer(store)

	body := strings.NewReader(`{"url":"https://github.com/mayks"}`)
	req := httptest.NewRequest("PUT", "/_/api/links/gh", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	url, _ := store.Get("gh")
	if url != "https://github.com/mayks" {
		t.Fatalf("link not updated: %q", url)
	}
}

func TestHandleAPIDelete(t *testing.T) {
	store := testStore(t)
	store.Add("gh", "https://github.com")
	handler := NewServer(store)

	req := httptest.NewRequest("DELETE", "/_/api/links/gh", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	_, ok := store.Get("gh")
	if ok {
		t.Fatal("expected link to be deleted")
	}
}

func TestHandleIndex(t *testing.T) {
	store := testStore(t)
	store.Add("gh", "https://github.com")
	handler := NewServer(store)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), "github.com") {
		t.Fatal("expected page to contain link URL")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -v -run TestHandle`
Expected: FAIL — `NewServer` not defined

**Step 3: Write the handlers implementation**

```go
// handlers.go
package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
)

//go:embed templates/index.html
var indexHTML string

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))

// NewServer creates an http.Handler with all routes configured.
func NewServer(store *LinkStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/_/api/links/", func(w http.ResponseWriter, r *http.Request) {
		phrase := strings.TrimPrefix(r.URL.Path, "/_/api/links/")
		if phrase == "" {
			http.Error(w, "phrase required", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodPut:
			var req struct {
				URL string `json:"url"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := store.Update(phrase, req.URL); err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)

		case http.MethodDelete:
			if err := store.Delete(phrase); err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/_/api/links", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Phrase string `json:"phrase"`
			URL    string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Phrase == "" || req.URL == "" {
			http.Error(w, "phrase and url required", http.StatusBadRequest)
			return
		}
		if err := store.Add(req.Phrase, req.URL); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			indexTmpl.Execute(w, store.All())
			return
		}

		// Redirect handler
		phrase := strings.TrimPrefix(r.URL.Path, "/")
		url, ok := store.Get(phrase)
		if !ok {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, url, http.StatusFound)
	})

	return mux
}
```

**Step 4: Run all tests to verify they pass**

Run: `go test -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add handlers.go handlers_test.go
git commit -m "feat: add HTTP handlers for redirect, API, and index page"
```

---

### Task 3: HTML Template

**Files:**
- Create: `templates/index.html`

**Step 1: Create the template directory**

Run: `mkdir -p templates`

**Step 2: Write the HTML template**

Create `templates/index.html` — a clean, minimal page with:
- A form to add new links (phrase + URL inputs + Add button)
- A table listing all links with edit/delete buttons
- Inline JavaScript for API calls (add, update, delete)
- Minimal embedded CSS for a clean look
- No external dependencies

The template receives `[]Link` as its data and renders each link as a table row.

Key behaviors:
- Add: POST to `/_/api/links`, reload page on success
- Delete: DELETE to `/_/api/links/{phrase}`, remove row on success
- Edit: Click edit, row becomes editable input, save calls PUT to `/_/api/links/{phrase}`

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go Links</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; max-width: 700px; margin: 40px auto; padding: 0 20px; color: #333; }
        h1 { margin-bottom: 24px; font-size: 24px; }
        .add-form { display: flex; gap: 8px; margin-bottom: 24px; }
        .add-form input { padding: 8px 12px; border: 1px solid #ccc; border-radius: 4px; font-size: 14px; }
        .add-form input[name="phrase"] { width: 150px; }
        .add-form input[name="url"] { flex: 1; }
        button { padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
        .btn-add { background: #2563eb; color: white; }
        .btn-add:hover { background: #1d4ed8; }
        .btn-edit { background: #f3f4f6; color: #333; }
        .btn-edit:hover { background: #e5e7eb; }
        .btn-delete { background: #fee2e2; color: #dc2626; }
        .btn-delete:hover { background: #fecaca; }
        .btn-save { background: #22c55e; color: white; }
        .btn-save:hover { background: #16a34a; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 10px 12px; text-align: left; border-bottom: 1px solid #e5e7eb; }
        th { font-weight: 600; color: #6b7280; font-size: 12px; text-transform: uppercase; }
        td.actions { white-space: nowrap; text-align: right; }
        td.actions button { margin-left: 4px; }
        .phrase-link { color: #2563eb; text-decoration: none; font-weight: 500; }
        .phrase-link:hover { text-decoration: underline; }
        .url-text { color: #6b7280; font-size: 13px; word-break: break-all; }
        .edit-input { width: 100%; padding: 6px 8px; border: 1px solid #2563eb; border-radius: 4px; font-size: 13px; }
        .empty { text-align: center; padding: 40px; color: #9ca3af; }
    </style>
</head>
<body>
    <h1>Go Links</h1>

    <form class="add-form" onsubmit="return addLink(event)">
        <input type="text" name="phrase" placeholder="phrase" required>
        <input type="url" name="url" placeholder="https://example.com" required>
        <button type="submit" class="btn-add">Add</button>
    </form>

    <table>
        <thead>
            <tr><th>Link</th><th>Destination</th><th></th></tr>
        </thead>
        <tbody id="links">
            {{range .}}
            <tr data-phrase="{{.Phrase}}">
                <td><a class="phrase-link" href="/{{.Phrase}}">go/{{.Phrase}}</a></td>
                <td class="url-cell"><span class="url-text">{{.URL}}</span></td>
                <td class="actions">
                    <button class="btn-edit" onclick="editLink(this)">Edit</button>
                    <button class="btn-delete" onclick="deleteLink('{{.Phrase}}')">Delete</button>
                </td>
            </tr>
            {{else}}
            <tr class="empty-row"><td colspan="3" class="empty">No links yet. Add one above.</td></tr>
            {{end}}
        </tbody>
    </table>

    <script>
        async function addLink(e) {
            e.preventDefault();
            const form = e.target;
            const phrase = form.phrase.value.trim();
            const url = form.url.value.trim();
            const res = await fetch('/_/api/links', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({phrase, url})
            });
            if (res.ok) location.reload();
            else alert(await res.text());
        }

        async function deleteLink(phrase) {
            if (!confirm('Delete go/' + phrase + '?')) return;
            const res = await fetch('/_/api/links/' + encodeURIComponent(phrase), {method: 'DELETE'});
            if (res.ok) location.reload();
            else alert(await res.text());
        }

        function editLink(btn) {
            const row = btn.closest('tr');
            const urlCell = row.querySelector('.url-cell');
            const currentUrl = urlCell.querySelector('.url-text').textContent;
            const phrase = row.dataset.phrase;

            urlCell.innerHTML = '<input class="edit-input" value="' + currentUrl + '">';
            const input = urlCell.querySelector('input');
            input.focus();

            const actions = row.querySelector('.actions');
            actions.innerHTML = '<button class="btn-save" onclick="saveLink(\'' + phrase + '\', this)">Save</button>';
        }

        async function saveLink(phrase, btn) {
            const row = btn.closest('tr');
            const newUrl = row.querySelector('.edit-input').value.trim();
            const res = await fetch('/_/api/links/' + encodeURIComponent(phrase), {
                method: 'PUT',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({url: newUrl})
            });
            if (res.ok) location.reload();
            else alert(await res.text());
        }
    </script>
</body>
</html>
```

**Step 3: Update handlers.go to use embed properly**

Add the `embed` import and the `//go:embed` directive. The `//go:embed` directive must be in a file in the same package. Update `handlers.go` to use `embed.FS` or a string embed.

Note: The `//go:embed` directive in the plan's Task 2 already references this file. It should compile now that the template exists.

**Step 4: Run all tests**

Run: `go test -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add templates/index.html
git commit -m "feat: add index page HTML template"
```

---

### Task 4: Main Entry Point

**Files:**
- Create: `main.go`

**Step 1: Write main.go**

```go
// main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	linksPath := filepath.Join(home, ".config", "golinks", "links.json")
	store, err := NewLinkStore(linksPath)
	if err != nil {
		log.Fatalf("Failed to load links: %v", err)
	}

	server := NewServer(store)

	addr := ":80"
	fmt.Printf("golinks server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatal(err)
	}
}
```

**Step 2: Build the binary**

Run: `go build -o golinks .`
Expected: Binary `golinks` created with no errors

**Step 3: Run all tests one more time**

Run: `go test -v`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add main.go
git commit -m "feat: add main entry point"
```

---

### Task 5: Deployment Setup

**Files:**
- Create: `install.sh` (convenience install script)

**Step 1: Write the install script**

```bash
#!/bin/bash
set -euo pipefail

echo "Building golinks..."
go build -o golinks .

echo "Installing binary to /usr/local/bin/golinks..."
sudo cp golinks /usr/local/bin/golinks

echo "Setting up /etc/hosts entry..."
if ! grep -q '127.0.0.1.*\bgo\b' /etc/hosts; then
    echo '127.0.0.1 go' | sudo tee -a /etc/hosts > /dev/null
    echo "Added '127.0.0.1 go' to /etc/hosts"
else
    echo "/etc/hosts already has 'go' entry"
fi

echo "Installing LaunchDaemon..."
sudo tee /Library/LaunchDaemons/com.golinks.server.plist > /dev/null <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.golinks.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/golinks</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/golinks.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/golinks.err</string>
</dict>
</plist>
PLIST

echo "Loading LaunchDaemon..."
sudo launchctl bootout system/com.golinks.server 2>/dev/null || true
sudo launchctl bootstrap system /Library/LaunchDaemons/com.golinks.server.plist

echo ""
echo "Done! golinks is running. Try: open http://go/"
```

**Step 2: Make it executable**

Run: `chmod +x install.sh`

**Step 3: Commit**

```bash
git add install.sh
git commit -m "feat: add install script with LaunchDaemon setup"
```

---

### Task 6: Build, Install, and Verify

**Step 1: Run the install script**

Run: `./install.sh`
Expected: Binary built, copied to /usr/local/bin, hosts entry added, LaunchDaemon loaded

**Step 2: Verify the server is running**

Run: `curl -s -o /dev/null -w '%{http_code}' http://go/`
Expected: `200`

**Step 3: Test adding a link via API**

Run: `curl -s -X POST http://go/_/api/links -H 'Content-Type: application/json' -d '{"phrase":"gh","url":"https://github.com"}'`
Expected: 201 response

**Step 4: Test the redirect**

Run: `curl -s -o /dev/null -w '%{http_code} %{redirect_url}' http://go/gh`
Expected: `302 https://github.com`

**Step 5: Open the UI in browser**

Run: `open http://go/`
Expected: Index page loads showing the "gh" link

**Step 6: Final commit with .gitignore**

```bash
echo "golinks" > .gitignore
git add .gitignore
git commit -m "chore: add .gitignore for built binary"
```
