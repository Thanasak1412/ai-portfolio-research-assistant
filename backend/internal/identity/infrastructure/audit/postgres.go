package audit

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
)

const (
	MaximumCorrelationIDLength = 128
	MaximumNetworkHashLength   = 96
	MaximumUserAgentLength     = 512
)

type Store interface {
	Append(context.Context, database.AuthenticationAuditRecord) error
}

type PostgresWriter struct{ store Store }

func NewPostgresWriter(store Store) *PostgresWriter { return &PostgresWriter{store: store} }

func (writer *PostgresWriter) Append(ctx context.Context, event application.AuditEvent) error {
	if writer == nil || writer.store == nil || validateEvent(event) != nil {
		return application.ErrAuditPersistence
	}
	eventID, err := uuid.NewRandom()
	if err != nil {
		return application.ErrAuditPersistence
	}
	record := database.AuthenticationAuditRecord{
		EventID:             eventID,
		OccurredAt:          event.OccurredAt,
		Action:              string(event.Action),
		Result:              string(event.Result),
		Severity:            string(event.Severity),
		ActorUserID:         userIDBytes(event.ActorUserID),
		CorrelationID:       event.CorrelationID,
		SessionID:           sessionIDBytes(event.SessionID),
		TokenFamilyID:       familyIDBytes(event.TokenFamilyID),
		NetworkIdentityHash: event.NetworkIdentityHash,
		UserAgent:           event.UserAgent,
	}
	if err := writer.store.Append(ctx, record); err != nil {
		return errors.Join(application.ErrAuditPersistence, errors.New("append audit record"))
	}
	return nil
}

func validateEvent(event application.AuditEvent) error {
	if event.OccurredAt.IsZero() || !allowedAction(event.Action) || !allowedResult(event.Result) || !allowedSeverity(event.Severity) {
		return application.ErrAuditPersistence
	}
	if strings.TrimSpace(event.CorrelationID) == "" || len(event.CorrelationID) > MaximumCorrelationIDLength || strings.ContainsAny(event.CorrelationID, "\r\n") {
		return application.ErrAuditPersistence
	}
	if len(event.NetworkIdentityHash) > MaximumNetworkHashLength || len(event.UserAgent) > MaximumUserAgentLength ||
		!safeMetadata(event.NetworkIdentityHash) || !safeMetadata(event.UserAgent) {
		return application.ErrAuditPersistence
	}
	if event.NetworkIdentityHash != "" && !strings.HasPrefix(event.NetworkIdentityHash, "ip_hmac_v1:") {
		return application.ErrAuditPersistence
	}
	if event.Action == application.AuditRefreshTokenReuse && event.Severity != application.AuditSeverityHigh {
		return application.ErrAuditPersistence
	}
	return nil
}

func safeMetadata(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func allowedAction(value application.AuditAction) bool {
	switch value {
	case application.AuditRegistrationSuccess, application.AuditRegistrationFailure,
		application.AuditLoginSuccess, application.AuditLoginFailure,
		application.AuditRefreshSuccess, application.AuditRefreshFailure,
		application.AuditLogout, application.AuditTokenFamilyRevocation,
		application.AuditRefreshTokenReuse, application.AuditDisabledAccountRejection:
		return true
	default:
		return false
	}
}

func allowedResult(value application.AuditResult) bool {
	return value == application.AuditResultSuccess || value == application.AuditResultFailure
}

func allowedSeverity(value application.AuditSeverity) bool {
	return value == application.AuditSeverityInfo || value == application.AuditSeverityWarning || value == application.AuditSeverityHigh
}

func userIDBytes(value *domain.UserID) *[16]byte {
	if value == nil {
		return nil
	}
	bytes := value.Bytes()
	return &bytes
}

func sessionIDBytes(value *domain.SessionID) *[16]byte {
	if value == nil {
		return nil
	}
	bytes := value.Bytes()
	return &bytes
}

func familyIDBytes(value *domain.TokenFamilyID) *[16]byte {
	if value == nil {
		return nil
	}
	bytes := value.Bytes()
	return &bytes
}

var _ application.AuditWriter = (*PostgresWriter)(nil)
