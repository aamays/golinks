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
