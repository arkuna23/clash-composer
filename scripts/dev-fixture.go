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
	addr := flag.String("addr", "127.0.0.1:8091", "fixture listen address")
	dir := flag.String("dir", ".", "directory containing source fixtures")
	flag.Parse()

	sourceDir, err := filepath.Abs(filepath.Join(*dir, "sources"))
	if err != nil {
		log.Fatalf("resolve source fixture directory: %v", err)
	}
	if info, err := os.Stat(sourceDir); err != nil {
		log.Fatalf("stat source fixture directory: %v", err)
	} else if !info.IsDir() {
		log.Fatalf("source fixture path is not a directory: %s", sourceDir)
	}

	mux := http.NewServeMux()
	mux.Handle("/sources/", http.StripPrefix("/sources/", http.FileServer(http.Dir(sourceDir))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "dev fixture not found", http.StatusNotFound)
	})

	server := &http.Server{Addr: *addr, Handler: mux}
	log.Printf("dev fixture listen start: addr=%s sources=%s", *addr, sourceDir)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(fmt.Errorf("dev fixture failed: %w", err))
	}
}
