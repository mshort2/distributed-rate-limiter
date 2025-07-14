package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    Server   ServerConfig
    Redis    RedisConfig
    RateLimit RateLimitConfig
}

type ServerConfig struct {
    Port         string
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
}

type RedisConfig struct {
    Host     string
    Port     string
    Password string
    DB       int
}

type RateLimitConfig struct {
    DefaultLimit  int
    DefaultWindow time.Duration
}

func Load() *Config {
    return &Config{
        Server: ServerConfig{
            Port:         getEnv("SERVER_PORT", "8080"),
            ReadTimeout:  getDuration("READ_TIMEOUT", 10*time.Second),
            WriteTimeout: getDuration("WRITE_TIMEOUT", 10*time.Second),
        },
        Redis: RedisConfig{
            Host:     getEnv("REDIS_HOST", "localhost"),
            Port:     getEnv("REDIS_PORT", "6379"),
            Password: getEnv("REDIS_PASSWORD", ""),
            DB:       getEnvInt("REDIS_DB", 0),
        },
        RateLimit: RateLimitConfig{
            DefaultLimit:  getEnvInt("DEFAULT_LIMIT", 100),
            DefaultWindow: getDuration("DEFAULT_WINDOW", time.Minute),
        },
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if i, err := strconv.Atoi(value); err == nil {
            return i
        }
    }
    return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if d, err := time.ParseDuration(value); err == nil {
            return d
        }
    }
    return defaultValue
}