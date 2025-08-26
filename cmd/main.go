package main

import (
	"fmt"
	"github.com/Kosench/go-url-shortener/internal/config"
	"log"
	"net/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "URL Shortener v0.1 - Coming Soon!")
	})

	log.Printf("Server starting on %s", cfg.GetServerAddress())
	log.Fatal(http.ListenAndServe(cfg.GetServerAddress(), nil))
}
