// links.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Link represents a single go-link entry.
type Link struct {
	Phrase string `json:"phrase"`
	URL    string `json:"url"`
}

// ScoredLink is a link with a relevance score for suggestion ranking.
type ScoredLink struct {
	Phrase   string
	URL      string
	Distance int
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

// Add creates a new link. Returns error if phrase already exists.
func (s *LinkStore) Add(phrase, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.links[phrase]; exists {
		return fmt.Errorf("phrase %q already exists", phrase)
	}

	s.links[phrase] = url
	if err := s.save(); err != nil {
		delete(s.links, phrase)
		return err
	}
	return nil
}

// Update modifies an existing link. Returns error if phrase not found.
func (s *LinkStore) Update(phrase, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, exists := s.links[phrase]
	if !exists {
		return fmt.Errorf("phrase %q not found", phrase)
	}

	s.links[phrase] = url
	if err := s.save(); err != nil {
		s.links[phrase] = old
		return err
	}
	return nil
}

// Delete removes a link. Returns error if phrase not found.
func (s *LinkStore) Delete(phrase string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, exists := s.links[phrase]
	if !exists {
		return fmt.Errorf("phrase %q not found", phrase)
	}

	delete(s.links, phrase)
	if err := s.save(); err != nil {
		s.links[phrase] = old
		return err
	}
	return nil
}

func (s *LinkStore) save() error {
	data, err := json.MarshalIndent(s.links, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal links: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	return os.Rename(tmp, s.path)
}

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
