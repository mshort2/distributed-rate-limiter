package test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/mshort2/distributed-rate-limiter/internal/server"
    "github.com/mshort2/distributed-rate-limiter/internal/config"
)

func TestRateLimiterEndpoints(t *testing.T) {
    cfg := config.Load()
    srv := server.NewServer(cfg)
    
    tests := []struct {
        name           string
        method         string
        path           string
        body           string
        expectedStatus int
    }{
        {
            name:           "health check",
            method:         "GET",
            path:           "/health",
            expectedStatus: http.StatusOK,
        },
        {
            name:           "rate limit check",
            method:         "POST",
            path:           "/check",
            body:           `{"client_id": "test-client"}`,
            expectedStatus: http.StatusOK,
        },
        {
            name:           "bad rate limit check",
            method:         "GET",
            path:           "/check",
            expectedStatus: http.StatusMethodNotAllowed,
        },
        {
            name:           "admin stats",
            method:         "GET",
            path:           "/admin/stats",
            expectedStatus: http.StatusOK,
        },
        {
            name:           "admin config",
            method:         "GET",
            path:           "/admin/config",
            expectedStatus: http.StatusOK,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var req *http.Request
            var err error
            
            if tt.body != "" {
                req, err = http.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
                req.Header.Set("Content-Type", "application/json")
            } else {
                req, err = http.NewRequest(tt.method, tt.path, nil)
            }
            
            if err != nil {
                t.Fatalf("Failed to create request: %v", err)
            }
            
            rr := httptest.NewRecorder()
            srv.ServeHTTP(rr, req)
            
            if rr.Code != tt.expectedStatus {
                t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
            }
        })
    }
}

func TestRateLimiterResponse(t *testing.T) {
    cfg := config.Load()
    srv := server.NewServer(cfg)
    
    body := `{"client_id": "test-client"}`
    req, err := http.NewRequest("POST", "/check", bytes.NewBufferString(body))
    if err != nil {
        t.Fatalf("Failed to create request: %v", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Client-ID", "test-client")
    
    rr := httptest.NewRecorder()
    srv.ServeHTTP(rr, req)
    
    if rr.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", rr.Code)
    }
    
    var response map[string]interface{}
    if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    
    // Check required fields
    requiredFields := []string{"allowed", "remaining", "reset_time", "client_id", "region"}
    for _, field := range requiredFields {
        if _, exists := response[field]; !exists {
            t.Errorf("Missing required field: %s", field)
        }
    }
}

func BenchmarkRateLimiter(b *testing.B) {
    cfg := config.Load()
    srv := server.NewServer(cfg)
    
    body := `{"client_id": "bench-client"}`
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            req, _ := http.NewRequest("POST", "/check", bytes.NewBufferString(body))
            req.Header.Set("Content-Type", "application/json")
            
            rr := httptest.NewRecorder()
            srv.ServeHTTP(rr, req)
        }
    })
}