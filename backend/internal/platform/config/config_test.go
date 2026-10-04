package config

import (
	"strings"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/worker"
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

func TestOutboxDeliveryDefaults(t *testing.T) {
	loaded, err := Load(mapLookup(map[string]string{"DATABASE_URL": "postgres://example.invalid/test"}))
	if err != nil {
		t.Fatal(err)
	}
	want := worker.DeliveryConfig{LeaseDuration: 60 * time.Second, RetryBaseDelay: 5 * time.Second, RetryMaxDelay: 5 * time.Minute, MaxDeliveryInvocations: 10, BatchSize: 50, PollInterval: 2 * time.Second}
	if loaded.OutboxDelivery != want {
		t.Fatalf("OUTBOX_DELIVERY-v1 defaults differ: %+v", loaded.OutboxDelivery)
	}
}

func TestOutboxDeliveryRejectsInvalidOverrides(t *testing.T) {
	for _, test := range []struct{ key, value string }{
		{"OUTBOX_LEASE_DURATION", "0s"}, {"OUTBOX_LEASE_DURATION", ""},
		{"OUTBOX_RETRY_BASE_DELAY", "-1s"}, {"OUTBOX_RETRY_MAX_DELAY", "4s"},
		{"OUTBOX_RETRY_MAX_DELAY", "invalid"}, {"OUTBOX_POLL_INTERVAL", "0s"},
		{"OUTBOX_MAX_DELIVERY_INVOCATIONS", "0"}, {"OUTBOX_MAX_DELIVERY_INVOCATIONS", "2147483648"},
		{"OUTBOX_MAX_DELIVERY_INVOCATIONS", "2147483647"},
		{"OUTBOX_BATCH_SIZE", "0"}, {"OUTBOX_BATCH_SIZE", "101"},
	} {
		t.Run(test.key+"="+test.value, func(t *testing.T) {
			_, err := Load(mapLookup(map[string]string{"DATABASE_URL": "postgres://example.invalid/test", test.key: test.value}))
			if err == nil || !strings.Contains(err.Error(), test.key) {
				t.Fatalf("expected field-specific rejection, got %v", err)
			}
		})
	}
}

func TestOutboxDeliveryAcceptsValidatedOverrides(t *testing.T) {
	loaded, err := Load(mapLookup(map[string]string{
		"DATABASE_URL": "postgres://example.invalid/test", "OUTBOX_LEASE_DURATION": "90s",
		"OUTBOX_RETRY_BASE_DELAY": "1s", "OUTBOX_RETRY_MAX_DELAY": "2s",
		"OUTBOX_MAX_DELIVERY_INVOCATIONS": "3", "OUTBOX_BATCH_SIZE": "100", "OUTBOX_POLL_INTERVAL": "4s",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := worker.DeliveryConfig{LeaseDuration: 90 * time.Second, RetryBaseDelay: time.Second, RetryMaxDelay: 2 * time.Second, MaxDeliveryInvocations: 3, BatchSize: 100, PollInterval: 4 * time.Second}
	if loaded.OutboxDelivery != want {
		t.Fatalf("overrides differ: %+v", loaded.OutboxDelivery)
	}
}
