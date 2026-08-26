package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/ju4n97/hclapi"
)

func main() {
	log.Println("Booting Hclapi API engine...")

	engine, err := hclapi.NewEngine(hclapi.Options{
		ManifestDir:  ".",
		StrictTyping: true,
	})
	if err != nil {
		log.Fatalf("Failed to initialize Hclapi engine: %v", err)
	}

	server := &http.Server{
		Addr:         ":8080",
		Handler:      engine.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Server successfully started on :8080")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
	}
}
