package main

import (
    "log"

    "github.com/mshort2/distributed-rate-limiter/internal/config"
    "github.com/mshort2/distributed-rate-limiter/internal/server"
)

func main() {
    cfg := config.Load()
    srv := server.NewServer(cfg)
    if err := srv.Start(); err != nil {
        log.Fatalf("Server shutdown error: %v", err)
    }
    log.Println("Server stopped")
}