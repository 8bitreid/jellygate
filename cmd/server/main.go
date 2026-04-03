package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/rmewborne/jellygate/internal/handler"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)

	slog.Info("jellygate starting", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
