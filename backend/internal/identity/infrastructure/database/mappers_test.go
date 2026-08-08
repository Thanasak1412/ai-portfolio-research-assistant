package database

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/database/sqlcgen"
)

func TestMappersFailClosedOnUnknownPersistedStates(t *testing.T) {
	now := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	userRow := sqlcgen.User{
		UserID: pgtype.UUID{Bytes: uuid.New(), Valid: true}, NormalizedEmail: "user@example.com",
		PasswordHash: "encoded-fixture", AccountStatus: "pending",
		CreatedAt: toPGTime(now), UpdatedAt: toPGTime(now),
	}
	if _, err := mapUserRow(userRow); !errors.Is(err, domain.ErrInvalidAccountStatus) {
		t.Fatalf("expected unknown account status to fail closed, got %v", err)
	}

	sessionRow := sqlcgen.RefreshSession{
		SessionID:     pgtype.UUID{Bytes: uuid.New(), Valid: true},
		TokenFamilyID: pgtype.UUID{Bytes: uuid.New(), Valid: true},
		UserID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
		TokenDigest:   make([]byte, domain.TokenDigestLength), SessionState: "unknown",
		CreatedAt: toPGTime(now), IdleExpiresAt: toPGTime(now.Add(24 * time.Hour)),
		AbsoluteExpiresAt: toPGTime(now.Add(48 * time.Hour)),
	}
	if _, err := mapRefreshSessionRow(sessionRow); !errors.Is(err, domain.ErrInvalidSessionState) {
		t.Fatalf("expected unknown session state to fail closed, got %v", err)
	}
}
