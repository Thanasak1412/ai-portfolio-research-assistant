package config

import (
	"strings"
	"testing"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	_, err := Load(mapLookup(map[string]string{}))
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("expected missing database error, got %v", err)
	}
}

func TestLoadUsesValidatedValues(t *testing.T) {
	loaded, err := Load(mapLookup(map[string]string{
		"APP_ENV":      "test",
		"HTTP_PORT":    "9090",
		"DATABASE_URL": "postgres://example.invalid/test",
	}))
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}
	if loaded.Environment != "test" || loaded.HTTPAddress() != "0.0.0.0:9090" {
		t.Fatalf("unexpected configuration: %+v", loaded)
	}
}

func mapLookup(values map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
