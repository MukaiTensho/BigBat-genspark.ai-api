package main

import (
	"fmt"
	"log"
	"net/http"

	"bigbat/internal/config"
	"bigbat/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("bigbat listening on %s", addr)
	if err = http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
