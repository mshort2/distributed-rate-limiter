package server

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
    "github.com/mshort2/distributed-rate-limiter/pkg/ratelimiter"
)

type Server struct {
    config    *config.Config
    server    *http.Server
    rl        *limiter.SlidingWindowLimiter
    startTime time.Time
}

func NewServer(cfg *config.Config) *Server {
    mux := http.NewServeMux()
    srv := &Server{
        config:    cfg,
        startTime: time.Now(),
    }

    mux.HandleFunc("/health", srv.healthHandler)
    mux.HandleFunc("/check", srv.rateLimitHandler)
    mux.HandleFunc("/admin/stats", srv.statsHandler)
    mux.HandleFunc("/admin/config", srv.configHandler)

    handler := middleware.Chain(
        middleware.RequestID,
        middleware.Logging,
        middleware.Recovery,
        middleware.CORS,
    )(mux)

    srv.server = &http.Server{
        Addr:         ":" + cfg.Server.Port,
        Handler:      handler,
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
    }

    rl, err := limiter.NewRateLimiter(cfg, cfg.RateLimit.DefaultLimit, cfg.RateLimit.DefaultWindow)
    if err != nil {
        log.Fatalf("Failed to create rate limiter: %v", err)
    }
    srv.rl = rl

    return srv
}

func (s *Server) rateLimitHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    clientID := extractClientID(r)
    requestID := r.Header.Get("X-Request-ID")

    response, err := s.rl.Allow(r.Context(), clientID, requestID)
    if err != nil {
        http.Error(w, "Rate limiter error", http.StatusInternalServerError)
        return
    }

    if response.Allowed {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusTooManyRequests)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func extractClientID(r *http.Request) string {
    if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
        return apiKey
    }
    if clientID := r.Header.Get("X-Client-ID"); clientID != "" {
        return clientID
    }
    if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
        return realIP
    }
    if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
        return forwarded
    }
    return r.RemoteAddr
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
    redisHealth := "healthy"
    if err := s.rl.Health(r.Context()); err != nil {
        redisHealth = "unhealthy"
    }

    health := map[string]string{
        "status":    redisHealth,
        "timestamp": time.Now().UTC().Format(time.RFC3339),
        "version":   "1.0.0",
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}

func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
    stats := map[string]interface{}{
        "requests_total":   1000,
        "requests_allowed": 950,
        "requests_denied":  50,
        "uptime_seconds":   time.Since(s.startTime).Seconds(),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
    safeCfg := map[string]interface{}{
        "server": map[string]interface{}{
            "port": s.config.Server.Port,
        },
        "rate_limit": map[string]interface{}{
            "default_limit":  s.config.RateLimit.DefaultLimit,
            "default_window": s.config.RateLimit.DefaultWindow.String(),
        },
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(safeCfg)
}

func (s *Server) Start() error {
    go func() {
        log.Printf("Server starting on port %s", s.config.Server.Port)
        if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed to start: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Server shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    return s.server.Shutdown(ctx)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.server.Handler.ServeHTTP(w, r)
}
