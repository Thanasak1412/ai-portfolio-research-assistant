package database

import (
	"context"

	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/infrastructure/database/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type ownershipReader struct{ queries *sqlcgen.Queries }

func NewOwnershipReader(tx pgx.Tx) application.OwnershipReader {
	return &ownershipReader{sqlcgen.New(tx)}
}
func (r *ownershipReader) GetPortfolio(ctx context.Context, principal identity.Principal, id domain.PortfolioID) (domain.Portfolio, error) {
	owner, ok := principal.UserID()
	if !ok {
		return domain.Portfolio{}, application.ErrUnauthenticated
	}
	if id.IsZero() {
		return domain.Portfolio{}, application.ErrPortfolioNotFound
	}
	row, err := r.queries.LockOwnedPortfolioForFinancialCommand(ctx, sqlcgen.LockOwnedPortfolioForFinancialCommandParams{PortfolioID: pgPortfolioID(id), OwnerUserID: pgOwnerID(owner)})
	if err != nil {
		return domain.Portfolio{}, mapPortfolioPersistenceError(err)
	}
	return mappedPortfolio(row)
}
