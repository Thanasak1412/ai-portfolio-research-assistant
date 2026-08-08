package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAccountStatusAndUserPredicates(t *testing.T) {
	if _, err := ParseAccountStatus("pending"); !errors.Is(err, ErrInvalidAccountStatus) {
		t.Fatalf("expected unknown status rejection, got %v", err)
	}
	for _, status := range []AccountStatus{AccountStatusActive, AccountStatusDisabled} {
		if parsed, err := ParseAccountStatus(string(status)); err != nil || parsed != status {
			t.Fatalf("parse %q: parsed=%q err=%v", status, parsed, err)
		}
	}

	now := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	active := mustUser(t, AccountStatusActive, now, nil)
	if !active.IsActive() || active.IsDisabled() {
		t.Fatal("active user predicates are inconsistent")
	}
	disabledAt := now.Add(time.Hour)
	disabled := mustUser(t, AccountStatusDisabled, now, &disabledAt)
	if disabled.IsActive() || !disabled.IsDisabled() {
		t.Fatal("disabled user predicates are inconsistent")
	}
	if got, ok := disabled.DisabledAt(); !ok || !got.Equal(disabledAt) {
		t.Fatalf("disabled timestamp: got=%v ok=%v", got, ok)
	}
}

func TestUserRejectsInconsistentDisabledState(t *testing.T) {
	now := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	id := mustUserID(t, "10000000-0000-4000-8000-000000000001")
	email, _ := NormalizeEmail("user@example.com")
	hash, _ := NewPasswordHash("encoded-secret-fixture")
	if _, err := NewUser(id, email, hash, AccountStatusDisabled, now, now, nil); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("expected disabled timestamp requirement, got %v", err)
	}
	disabledAt := now
	if _, err := NewUser(id, email, hash, AccountStatusActive, now, now, &disabledAt); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("expected active disabled timestamp rejection, got %v", err)
	}
}

func TestCredentialValuesAreRedactedFromFormatting(t *testing.T) {
	secret := "encoded-secret-fixture"
	hash, err := NewPasswordHash(secret)
	if err != nil {
		t.Fatal(err)
	}
	user := mustUser(t, AccountStatusActive, time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC), nil)
	for _, formatted := range []string{fmt.Sprint(hash), fmt.Sprintf("%#v", hash), fmt.Sprint(user), fmt.Sprintf("%#v", user)} {
		if strings.Contains(formatted, secret) {
			t.Fatalf("credential leaked through formatting: %q", formatted)
		}
	}
	serialized, err := json.Marshal(user)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), secret) {
		t.Fatalf("credential leaked through JSON serialization: %s", serialized)
	}
}

func mustUser(t *testing.T, status AccountStatus, now time.Time, disabledAt *time.Time) User {
	t.Helper()
	id := mustUserID(t, "10000000-0000-4000-8000-000000000001")
	email, err := NormalizeEmail("user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	hash, err := NewPasswordHash("encoded-secret-fixture")
	if err != nil {
		t.Fatal(err)
	}
	updatedAt := now
	if disabledAt != nil {
		updatedAt = *disabledAt
	}
	user, err := NewUser(id, email, hash, status, now, updatedAt, disabledAt)
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func mustUserID(t *testing.T, value string) UserID {
	t.Helper()
	id, err := NewUserID(uuid.MustParse(value))
	if err != nil {
		t.Fatal(err)
	}
	return id
}
