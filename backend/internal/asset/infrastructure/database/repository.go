package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/infrastructure/database/sqlcgen"
)

type PostgresAssetRepository struct{ queries *sqlcgen.Queries }

func NewPostgresAssetRepository(pool *pgxpool.Pool) *PostgresAssetRepository {
	return &PostgresAssetRepository{queries: sqlcgen.New(pool)}
}

func (repository *PostgresAssetRepository) GetByID(ctx context.Context, id domain.AssetID) (domain.Asset, error) {
	row, err := repository.queries.GetCanonicalAssetByID(ctx, pgAssetID(id))
	if err != nil {
		return domain.Asset{}, mapAssetPersistenceError(err)
	}
	asset, err := mapAsset(row)
	if err != nil {
		return domain.Asset{}, application.NewPersistenceError(application.ErrPersistenceFailure, err)
	}
	return asset, nil
}

func (repository *PostgresAssetRepository) Search(ctx context.Context, query application.SearchQuery) ([]domain.Asset, error) {
	params := sqlcgen.SearchCanonicalAssetsParams{Search: pgSearchText(query.Search, query.HasSearch), PageLimit: int32(query.Limit)}
	if query.AssetType != nil {
		params.AssetType = pgSearchText(string(*query.AssetType), true)
	}
	if query.After != nil {
		params.CursorSymbol = pgSearchText(query.After.Symbol, true)
		params.CursorExchange = pgSearchText(query.After.Exchange, true)
		params.CursorAssetID = pgAssetID(query.After.AssetID)
	}
	rows, err := repository.queries.SearchCanonicalAssets(ctx, params)
	if err != nil {
		return nil, mapAssetPersistenceError(err)
	}
	assets := make([]domain.Asset, 0, len(rows))
	for _, row := range rows {
		asset, mapErr := mapAsset(row)
		if mapErr != nil {
			return nil, application.NewPersistenceError(application.ErrPersistenceFailure, mapErr)
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

var _ application.Repository = (*PostgresAssetRepository)(nil)
