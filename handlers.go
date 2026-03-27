// handlers.go
package main

import (
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"
)

//go:embed templates/index.html
var indexHTML string

//go:embed templates/suggest.html
var suggestHTML string

//go:embed templates/static
var staticFS embed.FS

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

	staticSub, _ := fs.Sub(staticFS, "templates/static")
	mux.Handle("/_/static/", http.StripPrefix("/_/static/", http.FileServer(http.FS(staticSub))))

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
		if r.URL.Path == "/" || r.URL.Path == "/links" {
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
