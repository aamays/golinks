// links_test.go
package main

import (
	"fmt"
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

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"docs", "dcos", 2},
		{"mydocs", "docs", 2},
		{"kitten", "sitting", 3},
	}
	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
