package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database/sqlcgen"
)

const authenticationTransactionRollbackTimeout = 5 * time.Second

// AuthenticationAuditRecord is the allowlisted public persistence boundary for
// platform-owned Authentication audit evidence. It intentionally has no free-form
// metadata, credential, token, cookie, header, or key fields.
type AuthenticationAuditRecord struct {
	EventID             [16]byte
	OccurredAt          time.Time
	Action              string
	Result              string
	Severity            string
	ActorUserID         *[16]byte
	CorrelationID       string
	SessionID           *[16]byte
	TokenFamilyID       *[16]byte
	NetworkIdentityHash string
	UserAgent           string
}

type AuthenticationAuditStore struct{ queries *sqlcgen.Queries }

func NewAuthenticationAuditStore(database sqlcgen.DBTX) *AuthenticationAuditStore {
	return &AuthenticationAuditStore{queries: sqlcgen.New(database)}
}

func (store *AuthenticationAuditStore) Append(ctx context.Context, record AuthenticationAuditRecord) error {
	_, err := store.queries.AppendAuthenticationAuditEvent(ctx, sqlcgen.AppendAuthenticationAuditEventParams{
		AuditEventID:        pgUUID(record.EventID),
		OccurredAt:          pgTime(record.OccurredAt),
		Action:              record.Action,
		Result:              record.Result,
		Severity:            record.Severity,
		ActorUserID:         pgOptionalUUID(record.ActorUserID),
		CorrelationID:       record.CorrelationID,
		SessionID:           pgOptionalUUID(record.SessionID),
		TokenFamilyID:       pgOptionalUUID(record.TokenFamilyID),
		NetworkIdentityHash: pgOptionalText(record.NetworkIdentityHash),
		UserAgent:           pgOptionalText(record.UserAgent),
	})
	return err
}

// AuthenticationRateLimitTransaction exposes only the approved atomic rolling-
// window primitives. The generated sqlc package remains private to platform.
type AuthenticationRateLimitTransaction struct{ queries *sqlcgen.Queries }

type AuthenticationRateLimitTransactionOperations interface {
	AcquireAdvisoryLock(context.Context, int64) error
	DeleteExpiredForKey(context.Context, string, string, []byte, time.Time) (int64, error)
	CountActive(context.Context, string, string, []byte, time.Time) (int64, error)
	Insert(context.Context, AuthenticationRateLimitEvent) error
	EarliestActiveExpiry(context.Context, string, string, []byte, time.Time) (time.Time, error)
	ClearLoginEmailFailures(context.Context, string, []byte) (int64, error)
}

type AuthenticationRateLimitStore struct{ database sqlcgen.DBTX }

type AuthenticationRateLimitEvent struct {
	PolicyName    string
	PolicyVersion string
	DerivedKey    []byte
	OccurredAt    time.Time
	ExpiresAt     time.Time
}

func NewAuthenticationRateLimitStore(database sqlcgen.DBTX) *AuthenticationRateLimitStore {
	return &AuthenticationRateLimitStore{database: database}
}

func (store *AuthenticationRateLimitStore) WithinTransaction(
	ctx context.Context,
	operation func(AuthenticationRateLimitTransactionOperations) error,
) error {
	tx, err := beginTransaction(ctx, store.database)
	if err != nil {
		return err
	}
	defer func() {
		rollbackContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), authenticationTransactionRollbackTimeout)
		defer cancel()
		_ = tx.Rollback(rollbackContext)
	}()
	if err := operation(&AuthenticationRateLimitTransaction{queries: sqlcgen.New(tx)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (transaction *AuthenticationRateLimitTransaction) AcquireAdvisoryLock(ctx context.Context, key int64) error {
	return transaction.queries.AcquireAuthRateLimitAdvisoryLock(ctx, key)
}

func (transaction *AuthenticationRateLimitTransaction) DeleteExpiredForKey(
	ctx context.Context,
	policyName string,
	policyVersion string,
	derivedKey []byte,
	asOf time.Time,
) (int64, error) {
	return transaction.queries.DeleteExpiredAuthRateLimitEventsForKey(ctx, sqlcgen.DeleteExpiredAuthRateLimitEventsForKeyParams{
		PolicyName: policyName, PolicyVersion: policyVersion, DerivedKey: derivedKey, AsOf: pgTime(asOf),
	})
}

func (transaction *AuthenticationRateLimitTransaction) CountActive(
	ctx context.Context,
	policyName string,
	policyVersion string,
	derivedKey []byte,
	asOf time.Time,
) (int64, error) {
	return transaction.queries.CountActiveAuthRateLimitEvents(ctx, sqlcgen.CountActiveAuthRateLimitEventsParams{
		PolicyName: policyName, PolicyVersion: policyVersion, DerivedKey: derivedKey, AsOf: pgTime(asOf),
	})
}

func (transaction *AuthenticationRateLimitTransaction) Insert(
	ctx context.Context,
	event AuthenticationRateLimitEvent,
) error {
	_, err := transaction.queries.InsertAuthRateLimitEvent(ctx, sqlcgen.InsertAuthRateLimitEventParams{
		PolicyName: event.PolicyName, PolicyVersion: event.PolicyVersion, DerivedKey: event.DerivedKey,
		OccurredAt: pgTime(event.OccurredAt), ExpiresAt: pgTime(event.ExpiresAt),
	})
	return err
}

func (transaction *AuthenticationRateLimitTransaction) EarliestActiveExpiry(
	ctx context.Context,
	policyName string,
	policyVersion string,
	derivedKey []byte,
	asOf time.Time,
) (time.Time, error) {
	value, err := transaction.queries.GetEarliestActiveAuthRateLimitExpiry(ctx, sqlcgen.GetEarliestActiveAuthRateLimitExpiryParams{
		PolicyName: policyName, PolicyVersion: policyVersion, DerivedKey: derivedKey, AsOf: pgTime(asOf),
	})
	if err != nil {
		return time.Time{}, err
	}
	if !value.Valid {
		return time.Time{}, pgx.ErrNoRows
	}
	return value.Time, nil
}

func (transaction *AuthenticationRateLimitTransaction) ClearLoginEmailFailures(
	ctx context.Context,
	policyVersion string,
	derivedKey []byte,
) (int64, error) {
	return transaction.queries.ClearLoginEmailFailureEvents(ctx, sqlcgen.ClearLoginEmailFailureEventsParams{
		PolicyVersion: policyVersion, DerivedKey: derivedKey,
	})
}

func (store *AuthenticationRateLimitStore) CleanupExpired(ctx context.Context, asOf time.Time) (int64, error) {
	return sqlcgen.New(store.database).DeleteGloballyExpiredAuthRateLimitEvents(ctx, pgTime(asOf))
}

type transactionBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

func beginTransaction(ctx context.Context, database sqlcgen.DBTX) (pgx.Tx, error) {
	beginner, ok := database.(transactionBeginner)
	if !ok {
		return nil, pgx.ErrTxClosed
	}
	return beginner.Begin(ctx)
}

func pgUUID(value [16]byte) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: value != [16]byte{}}
}

func pgOptionalUUID(value *[16]byte) pgtype.UUID {
	if value == nil {
		return pgtype.UUID{}
	}
	return pgUUID(*value)
}

func pgTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}

func pgOptionalText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

var _ AuthenticationRateLimitTransactionOperations = (*AuthenticationRateLimitTransaction)(nil)
