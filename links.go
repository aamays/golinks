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
