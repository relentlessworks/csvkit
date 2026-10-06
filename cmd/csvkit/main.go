package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/relentlessworks/csvkit/internal/api"
	"github.com/relentlessworks/csvkit/internal/config"
)

func main() {
	cfg := config.Load()
	handler := api.NewHandler(cfg.NoAuth)
	mux := handler.Routes()

	log.Printf("csvkit starting on %s (no-auth=%v)", cfg.Addr, cfg.NoAuth)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// Ensure fmt is used (for potential future use)
var _ = fmt.Sprintf
