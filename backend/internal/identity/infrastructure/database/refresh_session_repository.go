package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/database/sqlcgen"
)

type PostgresRefreshSessionRepository struct{ queries *sqlcgen.Queries }

func NewPostgresRefreshSessionRepository(pool *pgxpool.Pool) *PostgresRefreshSessionRepository {
	return newPostgresRefreshSessionRepository(sqlcgen.New(pool))
}

func newPostgresRefreshSessionRepository(queries *sqlcgen.Queries) *PostgresRefreshSessionRepository {
	return &PostgresRefreshSessionRepository{queries: queries}
}

func (repository *PostgresRefreshSessionRepository) CreateInitial(ctx context.Context, session domain.RefreshSession) (domain.RefreshSession, error) {
	return repository.insertActive(ctx, session)
}

func (repository *PostgresRefreshSessionRepository) FindByID(ctx context.Context, id domain.SessionID) (domain.RefreshSession, error) {
	row, err := repository.queries.GetRefreshSessionByID(ctx, pgSessionID(id))
	if err != nil {
		return domain.RefreshSession{}, mapPersistenceError(err, application.ErrSessionNotFound)
	}
	return mapRefreshSessionRow(row)
}

func (repository *PostgresRefreshSessionRepository) ListFamily(ctx context.Context, familyID domain.TokenFamilyID) ([]domain.RefreshSession, error) {
	rows, err := repository.queries.ListRefreshTokenFamilyState(ctx, pgFamilyID(familyID))
	if err != nil {
		return nil, mapPersistenceError(err, application.ErrSessionNotFound)
	}
	return mapRefreshSessionRows(rows)
}

func (repository *PostgresRefreshSessionRepository) MarkExpired(ctx context.Context, asOf time.Time) ([]domain.SessionID, error) {
	rows, err := repository.queries.MarkExpiredRefreshSessions(ctx, toPGTime(asOf))
	if err != nil {
		return nil, mapPersistenceError(err, application.ErrSessionNotFound)
	}
	result := make([]domain.SessionID, 0, len(rows))
	for _, row := range rows {
		id, mapErr := sessionIDFromPG(row)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, id)
	}
	return result, nil
}

func (repository *PostgresRefreshSessionRepository) DeleteInactiveBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	count, err := repository.queries.DeleteInactiveRefreshSessionsBefore(ctx, toPGTime(cutoff))
	if err != nil {
		return 0, mapPersistenceError(err, application.ErrSessionNotFound)
	}
	return count, nil
}

func (repository *PostgresRefreshSessionRepository) insertActive(ctx context.Context, session domain.RefreshSession) (domain.RefreshSession, error) {
	if session.State() != domain.RefreshSessionStateActive {
		return domain.RefreshSession{}, domain.ErrSessionInactive
	}
	data := session.Data()
	row, err := repository.queries.InsertRefreshSessionGeneration(ctx, sqlcgen.InsertRefreshSessionGenerationParams{
		SessionID: pgSessionID(data.ID), TokenFamilyID: pgFamilyID(data.FamilyID), UserID: pgUserID(data.UserID),
		TokenDigest: data.TokenDigest.Bytes(), CreatedAt: toPGTime(data.CreatedAt), IdleExpiresAt: toPGTime(data.IdleExpiresAt),
		AbsoluteExpiresAt: toPGTime(data.AbsoluteExpiresAt), NetworkIdentityHash: pgOptionalText(data.NetworkIdentityHash),
		UserAgent: pgOptionalText(data.UserAgent),
	})
	if err != nil {
		return domain.RefreshSession{}, mapPersistenceError(err, application.ErrSessionNotFound)
	}
	return mapRefreshSessionRow(row)
}

type postgresRefreshSessionTransactionRepository struct {
	*PostgresRefreshSessionRepository
}

func (repository *postgresRefreshSessionTransactionRepository) LockByDigest(ctx context.Context, digest domain.TokenDigest) (domain.RefreshSession, error) {
	row, err := repository.queries.GetRefreshSessionByTokenDigestForUpdate(ctx, digest.Bytes())
	if err != nil {
		return domain.RefreshSession{}, mapPersistenceError(err, application.ErrSessionNotFound)
	}
	return mapRefreshSessionRow(row)
}

func (repository *postgresRefreshSessionTransactionRepository) MarkReplaced(ctx context.Context, session domain.RefreshSession) (domain.RefreshSession, error) {
	data := session.Data()
	if data.State != domain.RefreshSessionStateReplaced || data.ReplacementID == nil || data.ReplacedAt == nil {
		return domain.RefreshSession{}, domain.ErrInvalidReplacement
	}
	row, err := repository.queries.MarkActiveRefreshSessionReplaced(ctx, sqlcgen.MarkActiveRefreshSessionReplacedParams{
		ReplacementSessionID: pgSessionID(*data.ReplacementID), ReplacedAt: toPGTime(*data.ReplacedAt), SessionID: pgSessionID(data.ID),
	})
	if err != nil {
		if isNoRows(err) {
			return domain.RefreshSession{}, application.NewPersistenceError(application.ErrPersistenceConflict, err)
		}
		return domain.RefreshSession{}, mapPersistenceError(err, application.ErrSessionNotFound)
	}
	return mapRefreshSessionRow(row)
}

func (repository *postgresRefreshSessionTransactionRepository) InsertReplacement(ctx context.Context, session domain.RefreshSession) (domain.RefreshSession, error) {
	return repository.insertActive(ctx, session)
}

func (repository *postgresRefreshSessionTransactionRepository) RevokeFamily(
	ctx context.Context,
	familyID domain.TokenFamilyID,
	revokedAt time.Time,
	reason string,
) ([]domain.RefreshSession, error) {
	if familyID.IsZero() || revokedAt.IsZero() || reason == "" {
		return nil, domain.ErrInvalidRevocation
	}
	rows, err := repository.queries.RevokeRefreshTokenFamily(ctx, sqlcgen.RevokeRefreshTokenFamilyParams{
		RevokedAt: toPGTime(revokedAt), RevocationReason: pgOptionalText(reason), TokenFamilyID: pgFamilyID(familyID),
	})
	if err != nil {
		return nil, mapPersistenceError(err, application.ErrSessionNotFound)
	}
	return mapRefreshSessionRows(rows)
}

var _ application.RefreshSessionRepository = (*PostgresRefreshSessionRepository)(nil)
var _ application.RefreshSessionTransactionRepository = (*postgresRefreshSessionTransactionRepository)(nil)
