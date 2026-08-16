package database

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/infrastructure/database/sqlcgen"
)

func TestMapPortfolioFailsClosedForInvalidPersistedState(t *testing.T) {
	now := time.Date(2026, time.August, 16, 16, 0, 0, 0, time.UTC)
	row := sqlcgen.Portfolio{
		PortfolioID: pgtype.UUID{Bytes: uuid.New(), Valid: true},
		OwnerUserID: pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Name:        "Growth", BaseCurrency: "USD", Status: "UNKNOWN",
		CreatedAt: pgTime(now), UpdatedAt: pgTime(now),
	}
	if _, err := mapPortfolio(row); !errors.Is(err, domain.ErrInvalidPortfolioStatus) {
		t.Fatalf("unknown status error=%v", err)
	}
	row.Status = "ARCHIVED"
	if _, err := mapPortfolio(row); !errors.Is(err, domain.ErrInvalidPortfolio) {
		t.Fatalf("archived without timestamp error=%v", err)
	}
	row.Status = "ACTIVE"
	row.BaseCurrency = "THB"
	if _, err := mapPortfolio(row); !errors.Is(err, domain.ErrInvalidBaseCurrency) {
		t.Fatalf("non-USD error=%v", err)
	}
	row.BaseCurrency = "USD"
	row.OwnerUserID = pgtype.UUID{}
	if _, err := mapPortfolio(row); !errors.Is(err, identitydomain.ErrInvalidUserID) {
		t.Fatalf("invalid owner error=%v", err)
	}
}
