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
