# Embedding in Go

Hclapi is designed as a library-first runtime, implementing the standard library `http.Handler` interface.

## Minimal embedding

```go
package main

import (
	"log"
	"net/http"

	"github.com/ju4n97/hclapi"
)

func main() {
	// Initialize the Hclapi engine
	engine, err := hclapi.NewEngine(hclapi.Options{
		ManifestDir:  "./api",
		StrictTyping: true,
	})
	if err != nil {
		log.Fatalf("engine initialization failed: %v", err)
	}

	mux := http.NewServeMux()

	// Mount the Hclapi engine under a prefix
	mux.Handle("/api/v1/", engine.Handler())

	// Native Go routes operate side-by-side
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Fatal(http.ListenAndServe(":8080", mux))
}
```
