package database

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/infrastructure/database/sqlcgen"
)

func mapPortfolio(row sqlcgen.Portfolio) (domain.Portfolio, error) {
	id, err := portfolioIDFromPG(row.PortfolioID)
	if err != nil {
		return domain.Portfolio{}, err
	}
	ownerID, err := ownerIDFromPG(row.OwnerUserID)
	if err != nil {
		return domain.Portfolio{}, err
	}
	name, err := domain.NewPortfolioName(row.Name)
	if err != nil || name.String() != row.Name {
		return domain.Portfolio{}, domain.ErrInvalidPortfolioName
	}
	currency, err := domain.ParseBaseCurrency(row.BaseCurrency)
	if err != nil {
		return domain.Portfolio{}, err
	}
	status, err := domain.ParsePortfolioStatus(row.Status)
	if err != nil {
		return domain.Portfolio{}, err
	}
	createdAt, ok := requiredPortfolioTime(row.CreatedAt)
	if !ok {
		return domain.Portfolio{}, domain.ErrInvalidPortfolio
	}
	updatedAt, ok := requiredPortfolioTime(row.UpdatedAt)
	if !ok {
		return domain.Portfolio{}, domain.ErrInvalidPortfolio
	}
	return domain.RehydratePortfolio(id, ownerID, name, currency, status, optionalPortfolioTime(row.ArchivedAt), createdAt, updatedAt)
}

func portfolioIDFromPG(value pgtype.UUID) (domain.PortfolioID, error) {
	if !value.Valid {
		return domain.PortfolioID{}, domain.ErrInvalidPortfolioID
	}
	return domain.NewPortfolioID(uuid.UUID(value.Bytes))
}

func ownerIDFromPG(value pgtype.UUID) (identitydomain.UserID, error) {
	if !value.Valid {
		return identitydomain.UserID{}, identitydomain.ErrInvalidUserID
	}
	return identitydomain.NewUserID(uuid.UUID(value.Bytes))
}

func pgPortfolioID(value domain.PortfolioID) pgtype.UUID {
	return pgtype.UUID{Bytes: value.Bytes(), Valid: !value.IsZero()}
}

func pgOwnerID(value identitydomain.UserID) pgtype.UUID {
	return pgtype.UUID{Bytes: value.Bytes(), Valid: !value.IsZero()}
}

func pgTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}

func requiredPortfolioTime(value pgtype.Timestamptz) (time.Time, bool) {
	return value.Time, value.Valid
}

func optionalPortfolioTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
