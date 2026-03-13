# Dark Glassmorphism UI + Fuzzy Suggestions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign the golinks UI with dark glassmorphism theme and replace 404s with fuzzy link suggestions.

**Architecture:** Add a Levenshtein + substring fuzzy matcher to `LinkStore`, a new suggestion template, update the redirect handler to show suggestions instead of 404. Redesign `index.html` with dark glass aesthetic. All pure Go stdlib + CSS, no external deps.

**Tech Stack:** Go (stdlib only), embedded HTML/CSS templates

---

### Task 1: Levenshtein Distance + ScoredLink

**Files:**
- Modify: `links.go` (add after line 138)
- Test: `links_test.go` (add new tests)

**Step 1: Write the failing test for levenshtein**

Add to `links_test.go`:

```go
func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"docs", "dcos", 2},      // transposition
		{"mydocs", "docs", 2},    // prefix
		{"kitten", "sitting", 3}, // classic example
	}
	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestLevenshtein -v`
Expected: FAIL -- `levenshtein` undefined

**Step 3: Implement levenshtein in links.go**

Add to end of `links.go`:

```go
// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestLevenshtein -v`
Expected: PASS

**Step 5: Commit**

```bash
git add links.go links_test.go
git commit -m "feat: add levenshtein distance function"
```

---

### Task 2: Suggest Method on LinkStore

**Files:**
- Modify: `links.go` (add ScoredLink struct and Suggest method)
- Test: `links_test.go` (add Suggest tests)

**Step 1: Write the failing tests for Suggest**

Add to `links_test.go`:

```go
func TestSuggest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	store, _ := NewLinkStore(path)

	store.Add("docs", "https://docs.google.com")
	store.Add("drive", "https://drive.google.com")
	store.Add("mail", "https://mail.google.com")
	store.Add("meet", "https://meet.google.com")

	t.Run("substring match", func(t *testing.T) {
		results := store.Suggest("mydocs")
		if len(results) == 0 {
			t.Fatal("expected suggestions for 'mydocs'")
		}
		if results[0].Phrase != "docs" {
			t.Fatalf("expected 'docs' as top suggestion, got %q", results[0].Phrase)
		}
	})

	t.Run("edit distance match", func(t *testing.T) {
		results := store.Suggest("dcos")
		if len(results) == 0 {
			t.Fatal("expected suggestions for 'dcos'")
		}
		if results[0].Phrase != "docs" {
			t.Fatalf("expected 'docs' as top suggestion, got %q", results[0].Phrase)
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		results := store.Suggest("zzzzzzzzz")
		if len(results) != 0 {
			t.Fatalf("expected no suggestions, got %d", len(results))
		}
	})

	t.Run("max 5 results", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			store.Add(fmt.Sprintf("d%d", i), "https://example.com")
		}
		results := store.Suggest("d")
		if len(results) > 5 {
			t.Fatalf("expected at most 5 suggestions, got %d", len(results))
		}
	})
}
```

Note: add `"fmt"` to the imports in `links_test.go`.

**Step 2: Run test to verify it fails**

Run: `go test -run TestSuggest -v`
Expected: FAIL -- `store.Suggest` undefined

**Step 3: Implement ScoredLink and Suggest in links.go**

Add the `ScoredLink` struct after the `Link` struct (around line 17):

```go
// ScoredLink is a link with a relevance score for suggestion ranking.
type ScoredLink struct {
	Phrase   string
	URL      string
	Distance int
}
```

Add the `Suggest` method after the `All` method (around line 73):

```go
// Suggest returns up to 5 links that fuzzy-match the given phrase,
// using substring matching and Levenshtein edit distance.
func (s *LinkStore) Suggest(phrase string) []ScoredLink {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type candidate struct {
		phrase   string
		url      string
		distance int
		isSub    bool
	}

	var candidates []candidate
	lowerPhrase := strings.ToLower(phrase)

	for p, u := range s.links {
		lowerP := strings.ToLower(p)
		dist := levenshtein(lowerPhrase, lowerP)
		isSub := strings.Contains(lowerPhrase, lowerP) || strings.Contains(lowerP, lowerPhrase)

		// Include if substring match or edit distance is reasonable
		maxDist := len(phrase)
		if maxDist < 4 {
			maxDist = 4
		}
		if isSub || dist <= maxDist/2+1 {
			candidates = append(candidates, candidate{p, u, dist, isSub})
		}
	}

	// Sort: substring matches first, then by distance
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].isSub != candidates[j].isSub {
			return candidates[i].isSub
		}
		return candidates[i].distance < candidates[j].distance
	})

	limit := 5
	if len(candidates) < limit {
		limit = len(candidates)
	}

	results := make([]ScoredLink, limit)
	for i := 0; i < limit; i++ {
		results[i] = ScoredLink{
			Phrase:   candidates[i].phrase,
			URL:      candidates[i].url,
			Distance: candidates[i].distance,
		}
	}
	return results
}
```

Note: add `"strings"` to the imports in `links.go`.

**Step 4: Run test to verify it passes**

Run: `go test -run TestSuggest -v`
Expected: PASS

**Step 5: Run all tests to verify nothing broke**

Run: `go test -v`
Expected: All 15 tests PASS

**Step 6: Commit**

```bash
git add links.go links_test.go
git commit -m "feat: add fuzzy suggestion engine with substring and edit distance matching"
```

---

### Task 3: Suggestions Page Template

**Files:**
- Create: `templates/suggest.html`

**Step 1: Create the suggestions template**

Create `templates/suggest.html` with the dark glassmorphism theme. This template receives a struct with:
- `Query` (string): what the user typed
- `Suggestions` ([]ScoredLink): matched suggestions
- `AutoRedirect` (bool): whether to auto-redirect
- `RedirectPhrase` (string): the phrase to redirect to
- `RedirectURL` (string): the URL to redirect to

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>go/{{.Query}} - Not Found</title>
    {{if .AutoRedirect}}
    <meta http-equiv="refresh" content="2;url=/{{.RedirectPhrase}}">
    {{end}}
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
            background: #0a0a0f;
            color: #e2e8f0;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            overflow: hidden;
        }
        body::before {
            content: '';
            position: fixed;
            top: -50%;
            left: -50%;
            width: 200%;
            height: 200%;
            background: radial-gradient(circle at 30% 40%, rgba(99, 102, 241, 0.15) 0%, transparent 50%),
                        radial-gradient(circle at 70% 60%, rgba(139, 92, 246, 0.1) 0%, transparent 50%),
                        radial-gradient(circle at 50% 80%, rgba(59, 130, 246, 0.08) 0%, transparent 50%);
            z-index: 0;
        }
        .container {
            position: relative;
            z-index: 1;
            width: 100%;
            max-width: 560px;
            padding: 0 20px;
        }
        .glass-card {
            background: rgba(255, 255, 255, 0.05);
            backdrop-filter: blur(20px);
            -webkit-backdrop-filter: blur(20px);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 16px;
            padding: 40px;
        }
        .query-label {
            font-size: 13px;
            color: #64748b;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-bottom: 8px;
        }
        .query-phrase {
            font-size: 28px;
            font-weight: 700;
            color: #f1f5f9;
            margin-bottom: 6px;
        }
        .query-phrase span {
            color: #64748b;
            font-weight: 400;
        }
        .not-found-msg {
            color: #94a3b8;
            font-size: 14px;
            margin-bottom: 32px;
        }
        .redirect-msg {
            color: #a78bfa;
            font-size: 14px;
            margin-bottom: 24px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .redirect-msg .spinner {
            width: 16px;
            height: 16px;
            border: 2px solid rgba(167, 139, 250, 0.3);
            border-top-color: #a78bfa;
            border-radius: 50%;
            animation: spin 0.8s linear infinite;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
        .redirect-msg a {
            color: #64748b;
            text-decoration: none;
            margin-left: auto;
            font-size: 13px;
        }
        .redirect-msg a:hover { color: #94a3b8; }
        .suggestions-label {
            font-size: 12px;
            color: #64748b;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-bottom: 12px;
        }
        .suggestion {
            display: block;
            text-decoration: none;
            background: rgba(255, 255, 255, 0.03);
            border: 1px solid rgba(255, 255, 255, 0.06);
            border-radius: 10px;
            padding: 14px 18px;
            margin-bottom: 8px;
            transition: all 0.15s ease;
        }
        .suggestion:hover {
            background: rgba(255, 255, 255, 0.07);
            border-color: rgba(139, 92, 246, 0.3);
            transform: translateY(-1px);
        }
        .suggestion-phrase {
            font-weight: 600;
            color: #c4b5fd;
            font-size: 15px;
            margin-bottom: 4px;
        }
        .suggestion-url {
            color: #64748b;
            font-size: 12px;
            word-break: break-all;
        }
        .no-results {
            text-align: center;
            color: #475569;
            padding: 24px 0;
            font-size: 14px;
        }
        .home-link {
            display: inline-block;
            margin-top: 24px;
            color: #64748b;
            text-decoration: none;
            font-size: 13px;
            transition: color 0.15s;
        }
        .home-link:hover { color: #a78bfa; }
    </style>
</head>
<body>
    <div class="container">
        <div class="glass-card">
            <div class="query-label">Not Found</div>
            <div class="query-phrase"><span>go/</span>{{.Query}}</div>
            <div class="not-found-msg">This go-link doesn't exist yet.</div>

            {{if .AutoRedirect}}
            <div class="redirect-msg">
                <div class="spinner"></div>
                Redirecting to go/{{.RedirectPhrase}}...
                <a href="/" onclick="event.preventDefault(); window.stop(); this.closest('.redirect-msg').innerHTML='Cancelled.'; return false;">Cancel</a>
            </div>
            {{end}}

            {{if .Suggestions}}
            <div class="suggestions-label">Did you mean?</div>
            {{range .Suggestions}}
            <a class="suggestion" href="/{{.Phrase}}">
                <div class="suggestion-phrase">go/{{.Phrase}}</div>
                <div class="suggestion-url">{{.URL}}</div>
            </a>
            {{end}}
            {{else}}
            {{if not .AutoRedirect}}
            <div class="no-results">No similar links found.</div>
            {{end}}
            {{end}}

            <a class="home-link" href="/">&larr; Manage all links</a>
        </div>
    </div>
</body>
</html>
```

**Step 2: Commit**

```bash
git add templates/suggest.html
git commit -m "feat: add suggestions page template with glassmorphism theme"
```

---

### Task 4: Wire Suggestions Into Handler

**Files:**
- Modify: `handlers.go` (embed suggest template, update redirect handler)
- Test: `handlers_test.go` (update + add tests)

**Step 1: Write failing tests**

Update `TestHandleRedirectNotFound` and add new tests in `handlers_test.go`:

Replace the existing `TestHandleRedirectNotFound` (lines 40-51) with:

```go
func TestHandleRedirectNotFound(t *testing.T) {
	store := testStore(t)
	handler := NewServer(store)

	req := httptest.NewRequest("GET", "/nope", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Now returns 200 with suggestions page instead of 404
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Not Found") {
		t.Fatal("expected suggestions page with 'Not Found' text")
	}
}

func TestHandleSuggestionsWithMatches(t *testing.T) {
	store := testStore(t)
	store.Add("docs", "https://docs.google.com")
	handler := NewServer(store)

	req := httptest.NewRequest("GET", "/mydocs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "go/docs") {
		t.Fatal("expected suggestion 'go/docs' in response body")
	}
	if !strings.Contains(body, "Did you mean") {
		t.Fatal("expected 'Did you mean' text")
	}
}

func TestHandleAutoRedirect(t *testing.T) {
	store := testStore(t)
	store.Add("docs", "https://docs.google.com")
	handler := NewServer(store)

	// "doc" is edit distance 1 from "docs" -> should auto-redirect
	req := httptest.NewRequest("GET", "/doc", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Redirecting to") {
		t.Fatal("expected auto-redirect message")
	}
	if !strings.Contains(body, `meta http-equiv="refresh"`) {
		t.Fatal("expected meta refresh tag for auto-redirect")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run "TestHandleRedirectNotFound|TestHandleSuggestions|TestHandleAutoRedirect" -v`
Expected: FAIL -- tests reference new behavior not yet implemented

**Step 3: Update handlers.go**

Replace all of `handlers.go` with:

```go
// handlers.go
package main

import (
	_ "embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"
)

//go:embed templates/index.html
var indexHTML string

//go:embed templates/suggest.html
var suggestHTML string

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))
var suggestTmpl = template.Must(template.New("suggest").Parse(suggestHTML))

type suggestData struct {
	Query          string
	Suggestions    []ScoredLink
	AutoRedirect   bool
	RedirectPhrase string
	RedirectURL    string
}

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
			if req.URL == "" {
				http.Error(w, "url required", http.StatusBadRequest)
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
			if err := indexTmpl.Execute(w, store.All()); err != nil {
				log.Printf("template execute error: %v", err)
			}
			return
		}

		// Redirect handler
		phrase := strings.TrimPrefix(r.URL.Path, "/")
		url, ok := store.Get(phrase)
		if ok {
			http.Redirect(w, r, url, http.StatusFound)
			return
		}

		// Suggestion handler
		suggestions := store.Suggest(phrase)
		data := suggestData{
			Query:       phrase,
			Suggestions: suggestions,
		}

		// Auto-redirect if single high-confidence match (edit distance <= 2)
		if len(suggestions) == 1 && suggestions[0].Distance <= 2 {
			data.AutoRedirect = true
			data.RedirectPhrase = suggestions[0].Phrase
			data.RedirectURL = suggestions[0].URL
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := suggestTmpl.Execute(w, data); err != nil {
			log.Printf("suggest template execute error: %v", err)
		}
	})

	return mux
}
```

**Step 4: Run the new/updated tests**

Run: `go test -run "TestHandleRedirectNotFound|TestHandleSuggestions|TestHandleAutoRedirect" -v`
Expected: PASS

**Step 5: Run all tests**

Run: `go test -v`
Expected: All tests PASS (old + new)

**Step 6: Commit**

```bash
git add handlers.go handlers_test.go
git commit -m "feat: replace 404 with fuzzy suggestions page and auto-redirect"
```

---

### Task 5: Redesign Index Page with Dark Glassmorphism

**Files:**
- Modify: `templates/index.html` (full rewrite)

**Step 1: Rewrite index.html**

Replace all of `templates/index.html` with the dark glassmorphism design:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go Links</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
            background: #0a0a0f;
            color: #e2e8f0;
            min-height: 100vh;
            padding: 40px 20px;
        }
        body::before {
            content: '';
            position: fixed;
            top: -50%;
            left: -50%;
            width: 200%;
            height: 200%;
            background: radial-gradient(circle at 30% 40%, rgba(99, 102, 241, 0.15) 0%, transparent 50%),
                        radial-gradient(circle at 70% 60%, rgba(139, 92, 246, 0.1) 0%, transparent 50%),
                        radial-gradient(circle at 50% 80%, rgba(59, 130, 246, 0.08) 0%, transparent 50%);
            z-index: 0;
        }
        .container {
            position: relative;
            z-index: 1;
            max-width: 720px;
            margin: 0 auto;
        }
        h1 {
            font-size: 28px;
            font-weight: 700;
            margin-bottom: 28px;
            color: #f1f5f9;
        }
        h1 span {
            background: linear-gradient(135deg, #818cf8, #a78bfa);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }
        .glass-card {
            background: rgba(255, 255, 255, 0.05);
            backdrop-filter: blur(20px);
            -webkit-backdrop-filter: blur(20px);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 16px;
            padding: 24px;
            margin-bottom: 20px;
        }
        .add-form {
            display: flex;
            gap: 10px;
        }
        .add-form input {
            padding: 10px 14px;
            background: rgba(255, 255, 255, 0.05);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 10px;
            color: #e2e8f0;
            font-size: 14px;
            outline: none;
            transition: border-color 0.15s, box-shadow 0.15s;
        }
        .add-form input:focus {
            border-color: rgba(139, 92, 246, 0.5);
            box-shadow: 0 0 0 3px rgba(139, 92, 246, 0.15);
        }
        .add-form input::placeholder { color: #475569; }
        .add-form input[name="phrase"] { width: 160px; }
        .add-form input[name="url"] { flex: 1; }
        button {
            padding: 10px 18px;
            border: none;
            border-radius: 10px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 500;
            transition: all 0.15s;
        }
        .btn-add {
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            color: white;
        }
        .btn-add:hover {
            box-shadow: 0 4px 15px rgba(99, 102, 241, 0.4);
            transform: translateY(-1px);
        }
        .btn-edit {
            background: rgba(255, 255, 255, 0.06);
            color: #94a3b8;
            border: 1px solid rgba(255, 255, 255, 0.08);
        }
        .btn-edit:hover {
            background: rgba(255, 255, 255, 0.1);
            color: #e2e8f0;
        }
        .btn-delete {
            background: rgba(239, 68, 68, 0.1);
            color: #f87171;
            border: 1px solid rgba(239, 68, 68, 0.15);
        }
        .btn-delete:hover {
            background: rgba(239, 68, 68, 0.2);
        }
        .btn-save {
            background: linear-gradient(135deg, #22c55e, #16a34a);
            color: white;
        }
        .btn-save:hover {
            box-shadow: 0 4px 15px rgba(34, 197, 94, 0.3);
            transform: translateY(-1px);
        }
        table { width: 100%; border-collapse: collapse; }
        th {
            padding: 10px 14px;
            text-align: left;
            font-weight: 500;
            color: #64748b;
            font-size: 11px;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            border-bottom: 1px solid rgba(255, 255, 255, 0.06);
        }
        td {
            padding: 12px 14px;
            border-bottom: 1px solid rgba(255, 255, 255, 0.04);
        }
        tr:hover td {
            background: rgba(255, 255, 255, 0.02);
        }
        tr:last-child td { border-bottom: none; }
        td.actions { white-space: nowrap; text-align: right; }
        td.actions button { margin-left: 6px; }
        .phrase-link {
            color: #c4b5fd;
            text-decoration: none;
            font-weight: 600;
            font-size: 14px;
            transition: color 0.15s;
        }
        .phrase-link:hover { color: #e9d5ff; }
        .url-text {
            color: #64748b;
            font-size: 13px;
            word-break: break-all;
        }
        .edit-input {
            width: 100%;
            padding: 8px 10px;
            background: rgba(255, 255, 255, 0.05);
            border: 1px solid rgba(139, 92, 246, 0.4);
            border-radius: 8px;
            color: #e2e8f0;
            font-size: 13px;
            outline: none;
        }
        .edit-input:focus {
            box-shadow: 0 0 0 3px rgba(139, 92, 246, 0.15);
        }
        .empty {
            text-align: center;
            padding: 40px;
            color: #475569;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1><span>Go Links</span></h1>

        <div class="glass-card">
            <form class="add-form" onsubmit="return addLink(event)">
                <input type="text" name="phrase" placeholder="phrase" required>
                <input type="url" name="url" placeholder="https://example.com" required>
                <button type="submit" class="btn-add">Add</button>
            </form>
        </div>

        <div class="glass-card">
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
        </div>
    </div>

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

**Step 2: Run all tests to verify templates compile and render**

Run: `go test -v`
Expected: All tests PASS (templates are compiled at init time via `template.Must`)

**Step 3: Commit**

```bash
git add templates/index.html
git commit -m "feat: redesign index page with dark glassmorphism theme"
```

---

### Task 6: Final Verification

**Step 1: Run all tests**

Run: `go test -v`
Expected: All tests PASS

**Step 2: Build the binary**

Run: `go build -o golinks .`
Expected: Builds successfully with no errors

**Step 3: Commit (if any fixes needed)**

Only if fixes were applied. Otherwise skip.
