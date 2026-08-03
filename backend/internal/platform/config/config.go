package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type LookupFunc func(string) (string, bool)

type Config struct {
	Environment             string
	HTTPHost                string
	HTTPPort                int
	DatabaseURL             string
	DatabaseMaxConnections  int32
	DatabaseMinConnections  int32
	DatabaseConnectTimeout  time.Duration
	LogLevel                string
	ShutdownTimeout         time.Duration
	WorkerHeartbeatInterval time.Duration
}

func Load(lookup LookupFunc) (Config, error) {
	if lookup == nil {
		return Config{}, errors.New("configuration lookup is required")
	}

	config := Config{
		Environment:             valueOrDefault(lookup, "APP_ENV", "development"),
		HTTPHost:                valueOrDefault(lookup, "HTTP_HOST", "0.0.0.0"),
		DatabaseURL:             valueOrDefault(lookup, "DATABASE_URL", ""),
		LogLevel:                valueOrDefault(lookup, "LOG_LEVEL", "info"),
		WorkerHeartbeatInterval: 30 * time.Second,
	}

	var problems []string
	if !oneOf(config.Environment, "development", "test", "staging", "production") {
		problems = append(problems, "APP_ENV must be development, test, staging, or production")
	}
	if strings.TrimSpace(config.DatabaseURL) == "" {
		problems = append(problems, "DATABASE_URL is required")
	}
	if !oneOf(config.LogLevel, "debug", "info", "warn", "error") {
		problems = append(problems, "LOG_LEVEL must be debug, info, warn, or error")
	}

	config.HTTPPort = integer(lookup, "HTTP_PORT", 8080, 1, 65535, &problems)
	config.DatabaseMaxConnections = int32(integer(lookup, "DB_MAX_CONNS", 10, 1, 100, &problems))
	config.DatabaseMinConnections = int32(integer(lookup, "DB_MIN_CONNS", 1, 0, 100, &problems))
	config.DatabaseConnectTimeout = duration(lookup, "DB_CONNECT_TIMEOUT", 5*time.Second, &problems)
	config.ShutdownTimeout = duration(lookup, "SHUTDOWN_TIMEOUT", 10*time.Second, &problems)
	config.WorkerHeartbeatInterval = duration(lookup, "WORKER_HEARTBEAT_INTERVAL", 30*time.Second, &problems)

	if config.DatabaseMinConnections > config.DatabaseMaxConnections {
		problems = append(problems, "DB_MIN_CONNS cannot exceed DB_MAX_CONNS")
	}
	if len(problems) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %s", strings.Join(problems, "; "))
	}

	return config, nil
}

func (c Config) HTTPAddress() string {
	return c.HTTPHost + ":" + strconv.Itoa(c.HTTPPort)
}

func valueOrDefault(lookup LookupFunc, key, fallback string) string {
	if value, ok := lookup(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func integer(lookup LookupFunc, key string, fallback, minimum, maximum int, problems *[]string) int {
	raw := valueOrDefault(lookup, key, strconv.Itoa(fallback))
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		*problems = append(*problems, fmt.Sprintf("%s must be an integer between %d and %d", key, minimum, maximum))
		return fallback
	}
	return value
}

func duration(lookup LookupFunc, key string, fallback time.Duration, problems *[]string) time.Duration {
	raw := valueOrDefault(lookup, key, fallback.String())
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		*problems = append(*problems, key+" must be a positive duration")
		return fallback
	}
	return value
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
