package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/infrastructure/database/sqlcgen"
)

type PostgresPortfolioRepository struct{ queries *sqlcgen.Queries }

func NewPostgresPortfolioRepository(pool *pgxpool.Pool) *PostgresPortfolioRepository {
	return newPostgresPortfolioRepository(sqlcgen.New(pool))
}

func newPostgresPortfolioRepository(queries *sqlcgen.Queries) *PostgresPortfolioRepository {
	return &PostgresPortfolioRepository{queries: queries}
}

func (repository *PostgresPortfolioRepository) Create(ctx context.Context, portfolio domain.Portfolio) (domain.Portfolio, error) {
	row, err := repository.queries.CreatePortfolio(ctx, sqlcgen.CreatePortfolioParams{
		PortfolioID: pgPortfolioID(portfolio.ID()),
		OwnerUserID: pgOwnerID(portfolio.OwnerID()),
		Name:        portfolio.Name().String(),
		CreatedAt:   pgTime(portfolio.CreatedAt()),
	})
	if err != nil {
		return domain.Portfolio{}, mapPortfolioPersistenceError(err)
	}
	return mappedPortfolio(row)
}

func (repository *PostgresPortfolioRepository) FindOwnedByID(ctx context.Context, ownerID identitydomain.UserID, id domain.PortfolioID) (domain.Portfolio, error) {
	row, err := repository.queries.GetOwnedPortfolioByID(ctx, sqlcgen.GetOwnedPortfolioByIDParams{
		PortfolioID: pgPortfolioID(id), OwnerUserID: pgOwnerID(ownerID),
	})
	if err != nil {
		return domain.Portfolio{}, mapPortfolioPersistenceError(err)
	}
	return mappedPortfolio(row)
}

func (repository *PostgresPortfolioRepository) ListOwnedByStatus(ctx context.Context, ownerID identitydomain.UserID, status domain.PortfolioStatus) ([]domain.Portfolio, error) {
	rows, err := repository.queries.ListOwnedPortfoliosByStatus(ctx, sqlcgen.ListOwnedPortfoliosByStatusParams{
		OwnerUserID: pgOwnerID(ownerID), Status: string(status),
	})
	if err != nil {
		return nil, mapPortfolioPersistenceError(err)
	}
	portfolios := make([]domain.Portfolio, 0, len(rows))
	for _, row := range rows {
		portfolio, mapErr := mappedPortfolio(row)
		if mapErr != nil {
			return nil, application.NewPersistenceError(application.ErrPersistenceFailure, mapErr)
		}
		portfolios = append(portfolios, portfolio)
	}
	return portfolios, nil
}

func (repository *PostgresPortfolioRepository) UpdateOwnedActiveName(ctx context.Context, ownerID identitydomain.UserID, id domain.PortfolioID, name domain.PortfolioName, updatedAt time.Time) (domain.Portfolio, error) {
	row, err := repository.queries.UpdateOwnedActivePortfolioName(ctx, sqlcgen.UpdateOwnedActivePortfolioNameParams{
		Name: name.String(), UpdatedAt: pgTime(updatedAt),
		PortfolioID: pgPortfolioID(id), OwnerUserID: pgOwnerID(ownerID),
	})
	if err != nil {
		return domain.Portfolio{}, mapPortfolioPersistenceError(err)
	}
	return mappedPortfolio(row)
}

func (repository *PostgresPortfolioRepository) ArchiveOwnedActive(ctx context.Context, ownerID identitydomain.UserID, id domain.PortfolioID, archivedAt time.Time) (domain.Portfolio, error) {
	row, err := repository.queries.ArchiveOwnedActivePortfolio(ctx, sqlcgen.ArchiveOwnedActivePortfolioParams{
		ArchivedAt: pgTime(archivedAt), UpdatedAt: pgTime(archivedAt),
		PortfolioID: pgPortfolioID(id), OwnerUserID: pgOwnerID(ownerID),
	})
	if err != nil {
		return domain.Portfolio{}, mapPortfolioPersistenceError(err)
	}
	return mappedPortfolio(row)
}

var _ application.PortfolioRepository = (*PostgresPortfolioRepository)(nil)

func mappedPortfolio(row sqlcgen.Portfolio) (domain.Portfolio, error) {
	portfolio, err := mapPortfolio(row)
	if err != nil {
		return domain.Portfolio{}, application.NewPersistenceError(application.ErrPersistenceFailure, err)
	}
	return portfolio, nil
}
