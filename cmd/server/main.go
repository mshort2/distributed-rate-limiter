// cmd/server/main.go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/mshort2/distributed-rate-limiter/internal/config"
    "github.com/mshort2/distributed-rate-limiter/internal/middleware"
)

type RateLimitResponse struct {
    Allowed     bool   `json:"allowed"`
    Remaining   int    `json:"remaining"`
    ResetTime   int64  `json:"reset_time"`
    ClientID    string `json:"client_id"`
    Region      string `json:"region"`
    RequestID   string `json:"request_id"`
}

type Server struct {
    config *config.Config
    server *http.Server
}

func NewServer(cfg *config.Config) *Server {
    mux := http.NewServeMux()
    
    // Health check endpoint
    mux.HandleFunc("/health", healthHandler)
    
    // Rate limit check endpoint
    mux.HandleFunc("/check", rateLimitHandler)
    
    // Admin endpoints
    mux.HandleFunc("/admin/stats", statsHandler)
    mux.HandleFunc("/admin/config", configHandler)
    
    // Wrap with middleware
    handler := middleware.Chain(
        middleware.RequestID,
        middleware.Logging,
        middleware.Recovery,
        middleware.CORS,
    )(mux)
    
    server := &http.Server{
        Addr:         ":" + cfg.Server.Port,
        Handler:      handler,
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
    }
    
    return &Server{
        config: cfg,
        server: server,
    }
}

func rateLimitHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Extract client identifier
    clientID := extractClientID(r)
    requestID := r.Header.Get("X-Request-ID")
    
    // TODO: Replace with actual rate limiter
    // For now, implement simple mock with realistic behavior
    response := RateLimitResponse{
        Allowed:   true,
        Remaining: 95,
        ResetTime: time.Now().Add(time.Minute).Unix(),
        ClientID:  clientID,
        Region:    "us-east-1", // Will be dynamic later
        RequestID: requestID,
    }
    
    // Add realistic response time simulation
    time.Sleep(time.Millisecond * 2)
    
    // Set appropriate status code
    if response.Allowed {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusTooManyRequests)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func extractClientID(r *http.Request) string {
    // Priority order: API key > X-Client-ID header > IP address
    if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
        return apiKey
    }
    
    if clientID := r.Header.Get("X-Client-ID"); clientID != "" {
        return clientID
    }
    
    // Extract real IP (handle proxy headers)
    if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
        return realIP
    }
    
    if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
        return forwarded
    }
    
    return r.RemoteAddr
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    health := map[string]string{
        "status":    "healthy",
        "timestamp": time.Now().UTC().Format(time.RFC3339),
        "version":   "1.0.0",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
    stats := map[string]interface{}{
        "requests_total":   1000,
        "requests_allowed": 950,
        "requests_denied":  50,
        "uptime_seconds":   time.Since(time.Now()).Seconds(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

func configHandler(w http.ResponseWriter, r *http.Request) {
    cfg := config.Load()
    
    // Mask sensitive information
    safeCfg := map[string]interface{}{
        "server": map[string]interface{}{
            "port": cfg.Server.Port,
        },
        "rate_limit": map[string]interface{}{
            "default_limit":  cfg.RateLimit.DefaultLimit,
            "default_window": cfg.RateLimit.DefaultWindow.String(),
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(safeCfg)
}

func (s *Server) Start() error {
    // Start server in a goroutine
    go func() {
        log.Printf("Server starting on port %s", s.config.Server.Port)
        if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed to start: %v", err)
        }
    }()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Server shutting down...")
    
    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    return s.server.Shutdown(ctx)
}

func main() {
    cfg := config.Load()
    server := NewServer(cfg)
    
    if err := server.Start(); err != nil {
        log.Fatalf("Server shutdown error: %v", err)
    }
    
    log.Println("Server stopped")
}