package domain

import (
	"errors"
	"testing"
)

func TestPrincipalDefaultsToUnauthenticated(t *testing.T) {
	var absent Principal
	if absent.IsAuthenticated() {
		t.Fatal("zero principal must default to deny")
	}
	if _, ok := absent.UserID(); ok {
		t.Fatal("zero principal returned an authenticated user ID")
	}
	if _, err := NewPrincipal(UserID{}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected zero user rejection, got %v", err)
	}
}

func TestPrincipalUsesOnlyImmutableUserID(t *testing.T) {
	id := mustUserID(t, "10000000-0000-4000-8000-000000000002")
	principal, err := NewPrincipal(id)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := principal.UserID()
	if !ok || got != id || !principal.IsAuthenticated() {
		t.Fatalf("unexpected principal: id=%v authenticated=%v", got, principal.IsAuthenticated())
	}
}
