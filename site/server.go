// Package main provides a simple static file server for local testing.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	port := flag.Int("port", 8080, "Port to serve on")
	dir := flag.String("dir", ".", "Directory to serve")
	flag.Parse()

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("Failed to resolve directory: %v", err)
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		log.Fatalf("Directory does not exist: %s", absDir)
	}

	fs := http.FileServer(http.Dir(absDir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)

		// Redirect root to /site2skill-go/
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/site2skill-go/", http.StatusFound)
			return
		}

		// Handle /site2skill-go/ prefix
		prefix := "/site2skill-go"
		if len(r.URL.Path) >= len(prefix) && r.URL.Path[:len(prefix)] == prefix {
			http.StripPrefix(prefix, fs).ServeHTTP(w, r)
			return
		}

		// Fallback (e.g. for /favicon.ico if not strictly prefixed, or direct access)
		fs.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("🌐 Serving %s at http://localhost%s\n", absDir, addr)
	fmt.Println("Press Ctrl+C to stop")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
