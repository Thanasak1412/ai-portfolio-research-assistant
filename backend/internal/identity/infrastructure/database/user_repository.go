package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/database/sqlcgen"
)

type PostgresUserRepository struct{ queries *sqlcgen.Queries }

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return newPostgresUserRepository(sqlcgen.New(pool))
}

func newPostgresUserRepository(queries *sqlcgen.Queries) *PostgresUserRepository {
	return &PostgresUserRepository{queries: queries}
}

func (repository *PostgresUserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	if !user.IsActive() {
		return domain.User{}, domain.ErrInvalidUser
	}
	row, err := repository.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		UserID: pgUserID(user.ID()), NormalizedEmail: user.Email().String(),
		PasswordHash: user.PasswordHash().Encoded(), CreatedAt: toPGTime(user.CreatedAt()),
	})
	if err != nil {
		return domain.User{}, mapPersistenceError(err, application.ErrUserNotFound)
	}
	return mapUserRow(row)
}

func (repository *PostgresUserRepository) FindByNormalizedEmail(ctx context.Context, email domain.NormalizedEmail) (domain.User, error) {
	row, err := repository.queries.GetUserByNormalizedEmail(ctx, email.String())
	if err != nil {
		return domain.User{}, mapPersistenceError(err, application.ErrUserNotFound)
	}
	return mapUserRow(row)
}

func (repository *PostgresUserRepository) FindByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	row, err := repository.queries.GetUserByID(ctx, pgUserID(id))
	if err != nil {
		return domain.User{}, mapPersistenceError(err, application.ErrUserNotFound)
	}
	return mapUserRow(row)
}

func (repository *PostgresUserRepository) CompareAndSwapPasswordHash(
	ctx context.Context,
	id domain.UserID,
	expected domain.PasswordHash,
	replacement domain.PasswordHash,
	updatedAt time.Time,
) (domain.User, error) {
	if id.IsZero() || expected.IsZero() || replacement.IsZero() || updatedAt.IsZero() {
		return domain.User{}, application.ErrPersistenceConflict
	}
	row, err := repository.queries.UpdatePasswordHashCompareAndSwap(ctx, sqlcgen.UpdatePasswordHashCompareAndSwapParams{
		NewPasswordHash: replacement.Encoded(), UpdatedAt: toPGTime(updatedAt),
		UserID: pgUserID(id), ExpectedPasswordHash: expected.Encoded(),
	})
	if err != nil {
		if isNoRows(err) {
			return domain.User{}, application.NewPersistenceError(application.ErrPersistenceConflict, err)
		}
		return domain.User{}, mapPersistenceError(err, application.ErrUserNotFound)
	}
	return mapUserRow(row)
}

var _ application.UserRepository = (*PostgresUserRepository)(nil)
