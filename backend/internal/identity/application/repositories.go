package application

import (
	"context"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

type UserRepository interface {
	Create(context.Context, domain.User) (domain.User, error)
	FindByNormalizedEmail(context.Context, domain.NormalizedEmail) (domain.User, error)
	FindByID(context.Context, domain.UserID) (domain.User, error)
	CompareAndSwapPasswordHash(context.Context, domain.UserID, domain.PasswordHash, domain.PasswordHash, time.Time) (domain.User, error)
}

type RefreshSessionRepository interface {
	CreateInitial(context.Context, domain.RefreshSession) (domain.RefreshSession, error)
	FindByID(context.Context, domain.SessionID) (domain.RefreshSession, error)
	ListFamily(context.Context, domain.TokenFamilyID) ([]domain.RefreshSession, error)
	MarkExpired(context.Context, time.Time) ([]domain.SessionID, error)
	DeleteInactiveBefore(context.Context, time.Time) (int64, error)
}

// RefreshSessionTransactionRepository exposes lifecycle changes that must use
// the same PostgreSQL transaction. LockByDigest is intentionally absent from
// the pool-backed RefreshSessionRepository.
type RefreshSessionTransactionRepository interface {
	RefreshSessionRepository
	LockByDigest(context.Context, domain.TokenDigest) (domain.RefreshSession, error)
	MarkReplaced(context.Context, domain.RefreshSession) (domain.RefreshSession, error)
	InsertReplacement(context.Context, domain.RefreshSession) (domain.RefreshSession, error)
	// RevokeFamily is the persistence primitive for both current-session logout
	// and token-reuse response. One family represents one browser login session.
	RevokeFamily(context.Context, domain.TokenFamilyID, time.Time, string) ([]domain.RefreshSession, error)
}

type TransactionRepositories interface {
	Users() UserRepository
	RefreshSessions() RefreshSessionTransactionRepository
	Audit() AuditWriter
}

type Transactor interface {
	WithinTransaction(context.Context, func(context.Context, TransactionRepositories) error) error
}
