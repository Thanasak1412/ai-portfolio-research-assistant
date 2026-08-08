package domain

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRefreshSessionStateAndExpiry(t *testing.T) {
	session, now := activeSessionFixture(t)
	if !session.IsActiveAt(now) || session.IsExpiredAt(now) {
		t.Fatal("new active session should be active")
	}
	if session.IsActiveAt(session.IdleExpiresAt()) || !session.IsExpiredAt(session.IdleExpiresAt()) {
		t.Fatal("idle expiry boundary must be expired")
	}
	if session.IsActiveAt(session.AbsoluteExpiresAt()) || !session.IsExpiredAt(session.AbsoluteExpiresAt()) {
		t.Fatal("absolute expiry boundary must be expired")
	}
	if _, err := ParseRefreshSessionState("unknown"); !errors.Is(err, ErrInvalidSessionState) {
		t.Fatalf("expected unknown state rejection, got %v", err)
	}
	expiredData := session.Data()
	expiredData.State = RefreshSessionStateExpired
	expired, err := NewRefreshSession(expiredData)
	if err != nil || !expired.IsExpiredAt(now) || expired.IsActiveAt(now) {
		t.Fatalf("persisted expired state: active=%v expired=%v err=%v", expired.IsActiveAt(now), expired.IsExpiredAt(now), err)
	}
}

func TestPlanReplacementPreservesFamilyUserAndAbsoluteExpiry(t *testing.T) {
	session, now := activeSessionFixture(t)
	replacementID := mustSessionID(t, "20000000-0000-4000-8000-000000000002")
	replaced, replacement, err := session.PlanReplacement(ReplacementInput{
		SessionID: replacementID, TokenDigest: digestFixture(t, 2), CreatedAt: now.Add(time.Minute),
		IdleExpiresAt: now.Add(24 * time.Hour), NetworkIdentityHash: "network-hash", UserAgent: "browser",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !replaced.IsReplaced() || replacement.State() != RefreshSessionStateActive {
		t.Fatalf("unexpected states: original=%q replacement=%q", replaced.State(), replacement.State())
	}
	if replacement.FamilyID() != session.FamilyID() || replacement.UserID() != session.UserID() {
		t.Fatal("replacement changed family or user")
	}
	if !replacement.AbsoluteExpiresAt().Equal(session.AbsoluteExpiresAt()) {
		t.Fatal("replacement changed family absolute expiry")
	}
	data := replaced.Data()
	if data.ReplacementID == nil || *data.ReplacementID != replacementID || data.ReplacedAt == nil {
		t.Fatal("original replacement link was not planned")
	}
}

func TestPlanReplacementRejectsSelfAndInactiveSessions(t *testing.T) {
	session, now := activeSessionFixture(t)
	input := ReplacementInput{
		SessionID: session.ID(), TokenDigest: digestFixture(t, 2), CreatedAt: now.Add(time.Minute),
		IdleExpiresAt: now.Add(24 * time.Hour),
	}
	if _, _, err := session.PlanReplacement(input); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("expected self replacement rejection, got %v", err)
	}

	input.SessionID = mustSessionID(t, "20000000-0000-4000-8000-000000000003")
	revoked, err := session.Revoke(now.Add(time.Minute), "logout")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := revoked.PlanReplacement(input); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("expected revoked replacement rejection, got %v", err)
	}

	replaced, _, err := session.PlanReplacement(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := replaced.PlanReplacement(ReplacementInput{
		SessionID:   mustSessionID(t, "20000000-0000-4000-8000-000000000004"),
		TokenDigest: digestFixture(t, 4), CreatedAt: now.Add(2 * time.Minute), IdleExpiresAt: now.Add(24 * time.Hour),
	}); !errors.Is(err, ErrSessionReplaced) {
		t.Fatalf("expected replaced rejection, got %v", err)
	}
}

func TestRefreshSessionRevocationAndSensitiveFormatting(t *testing.T) {
	session, now := activeSessionFixture(t)
	revoked, err := session.Revoke(now.Add(time.Minute), "logout")
	if err != nil {
		t.Fatal(err)
	}
	if !revoked.IsRevoked() || revoked.Data().RevocationReason != "logout" {
		t.Fatal("revocation state was not represented")
	}
	digestBytes := session.TokenDigest().Bytes()
	digestText := fmt.Sprint(digestBytes)
	for _, formatted := range []string{fmt.Sprint(session.TokenDigest()), fmt.Sprintf("%#v", session.TokenDigest()), fmt.Sprint(session)} {
		if strings.Contains(formatted, digestText) {
			t.Fatalf("token digest leaked through formatting: %q", formatted)
		}
	}
}

func activeSessionFixture(t *testing.T) (RefreshSession, time.Time) {
	t.Helper()
	now := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	session, err := NewActiveRefreshSession(
		mustSessionID(t, "20000000-0000-4000-8000-000000000001"),
		mustFamilyID(t, "30000000-0000-4000-8000-000000000001"),
		mustUserID(t, "10000000-0000-4000-8000-000000000001"),
		digestFixture(t, 1), now, now.Add(30*24*time.Hour), now.Add(90*24*time.Hour), "network-hash", "browser",
	)
	if err != nil {
		t.Fatal(err)
	}
	return session, now
}

func mustSessionID(t *testing.T, value string) SessionID {
	t.Helper()
	id, err := NewSessionID(uuid.MustParse(value))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustFamilyID(t *testing.T, value string) TokenFamilyID {
	t.Helper()
	id, err := NewTokenFamilyID(uuid.MustParse(value))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func digestFixture(t *testing.T, seed byte) TokenDigest {
	t.Helper()
	value := make([]byte, TokenDigestLength)
	value[len(value)-1] = seed
	digest, err := NewTokenDigest(value)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
