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
		if !ok {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, url, http.StatusFound)
	})

	return mux
}
