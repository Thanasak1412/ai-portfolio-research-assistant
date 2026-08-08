package database

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/database/sqlcgen"
)

func mapUserRow(row sqlcgen.User) (domain.User, error) {
	id, err := userIDFromPG(row.UserID)
	if err != nil {
		return domain.User{}, err
	}
	email, err := domain.NormalizeEmail(row.NormalizedEmail)
	if err != nil || email.String() != row.NormalizedEmail {
		return domain.User{}, domain.ErrInvalidEmail
	}
	passwordHash, err := domain.NewPasswordHash(row.PasswordHash)
	if err != nil {
		return domain.User{}, err
	}
	status, err := domain.ParseAccountStatus(row.AccountStatus)
	if err != nil {
		return domain.User{}, err
	}
	createdAt, ok := requiredTime(row.CreatedAt)
	if !ok {
		return domain.User{}, domain.ErrInvalidUser
	}
	updatedAt, ok := requiredTime(row.UpdatedAt)
	if !ok {
		return domain.User{}, domain.ErrInvalidUser
	}
	disabledAt := optionalTime(row.DisabledAt)
	return domain.NewUser(id, email, passwordHash, status, createdAt, updatedAt, disabledAt)
}

func mapRefreshSessionRow(row sqlcgen.RefreshSession) (domain.RefreshSession, error) {
	id, err := sessionIDFromPG(row.SessionID)
	if err != nil {
		return domain.RefreshSession{}, err
	}
	familyID, err := familyIDFromPG(row.TokenFamilyID)
	if err != nil {
		return domain.RefreshSession{}, err
	}
	userID, err := userIDFromPG(row.UserID)
	if err != nil {
		return domain.RefreshSession{}, err
	}
	digest, err := domain.NewTokenDigest(row.TokenDigest)
	if err != nil {
		return domain.RefreshSession{}, err
	}
	state, err := domain.ParseRefreshSessionState(row.SessionState)
	if err != nil {
		return domain.RefreshSession{}, err
	}
	createdAt, ok := requiredTime(row.CreatedAt)
	if !ok {
		return domain.RefreshSession{}, domain.ErrInvalidRefreshSession
	}
	idleExpiresAt, ok := requiredTime(row.IdleExpiresAt)
	if !ok {
		return domain.RefreshSession{}, domain.ErrInvalidRefreshSession
	}
	absoluteExpiresAt, ok := requiredTime(row.AbsoluteExpiresAt)
	if !ok {
		return domain.RefreshSession{}, domain.ErrInvalidRefreshSession
	}
	var replacementID *domain.SessionID
	if row.ReplacementSessionID.Valid {
		value, mapErr := sessionIDFromPG(row.ReplacementSessionID)
		if mapErr != nil {
			return domain.RefreshSession{}, mapErr
		}
		replacementID = &value
	}
	return domain.NewRefreshSession(domain.RefreshSessionData{
		ID: id, FamilyID: familyID, UserID: userID, TokenDigest: digest, State: state,
		ReplacementID: replacementID, CreatedAt: createdAt, ReplacedAt: optionalTime(row.ReplacedAt),
		IdleExpiresAt: idleExpiresAt, AbsoluteExpiresAt: absoluteExpiresAt,
		RevokedAt: optionalTime(row.RevokedAt), RevocationReason: optionalText(row.RevocationReason),
		NetworkIdentityHash: optionalText(row.NetworkIdentityHash), UserAgent: optionalText(row.UserAgent),
	})
}

func mapRefreshSessionRows(rows []sqlcgen.RefreshSession) ([]domain.RefreshSession, error) {
	result := make([]domain.RefreshSession, 0, len(rows))
	for _, row := range rows {
		session, err := mapRefreshSessionRow(row)
		if err != nil {
			return nil, err
		}
		result = append(result, session)
	}
	return result, nil
}

func userIDFromPG(value pgtype.UUID) (domain.UserID, error) {
	if !value.Valid {
		return domain.UserID{}, domain.ErrInvalidUserID
	}
	return domain.NewUserID(uuid.UUID(value.Bytes))
}

func sessionIDFromPG(value pgtype.UUID) (domain.SessionID, error) {
	if !value.Valid {
		return domain.SessionID{}, domain.ErrInvalidSessionID
	}
	return domain.NewSessionID(uuid.UUID(value.Bytes))
}

func familyIDFromPG(value pgtype.UUID) (domain.TokenFamilyID, error) {
	if !value.Valid {
		return domain.TokenFamilyID{}, domain.ErrInvalidTokenFamilyID
	}
	return domain.NewTokenFamilyID(uuid.UUID(value.Bytes))
}

func pgUserID(value domain.UserID) pgtype.UUID {
	return pgtype.UUID{Bytes: value.Bytes(), Valid: !value.IsZero()}
}

func pgSessionID(value domain.SessionID) pgtype.UUID {
	return pgtype.UUID{Bytes: value.Bytes(), Valid: !value.IsZero()}
}

func pgFamilyID(value domain.TokenFamilyID) pgtype.UUID {
	return pgtype.UUID{Bytes: value.Bytes(), Valid: !value.IsZero()}
}

func toPGTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}

func requiredTime(value pgtype.Timestamptz) (time.Time, bool) {
	return value.Time, value.Valid
}

func optionalTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func optionalText(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func pgOptionalText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}
