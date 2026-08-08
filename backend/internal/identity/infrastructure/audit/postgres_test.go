package audit

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	platformdatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
)

type captureStore struct {
	records []platformdatabase.AuthenticationAuditRecord
	err     error
}

func (store *captureStore) Append(_ context.Context, record platformdatabase.AuthenticationAuditRecord) error {
	store.records = append(store.records, record)
	return store.err
}

func TestPostgresWriterMapsEveryApprovedActionToAllowlistedRecord(t *testing.T) {
	userID, _ := domain.NewUserID(uuid.New())
	sessionID, _ := domain.NewSessionID(uuid.New())
	familyID, _ := domain.NewTokenFamilyID(uuid.New())
	store := &captureStore{}
	writer := NewPostgresWriter(store)
	actions := []application.AuditAction{
		application.AuditRegistrationSuccess, application.AuditRegistrationFailure,
		application.AuditLoginSuccess, application.AuditLoginFailure,
		application.AuditRefreshSuccess, application.AuditRefreshFailure,
		application.AuditLogout, application.AuditTokenFamilyRevocation,
		application.AuditRefreshTokenReuse, application.AuditDisabledAccountRejection,
	}
	for _, action := range actions {
		severity := application.AuditSeverityInfo
		if action == application.AuditRefreshTokenReuse {
			severity = application.AuditSeverityHigh
		}
		event := application.AuditEvent{
			OccurredAt: time.Now().UTC(), Action: action, Result: application.AuditResultSuccess,
			Severity: severity, ActorUserID: &userID, CorrelationID: "corr-safe",
			SessionID: &sessionID, TokenFamilyID: &familyID,
			NetworkIdentityHash: "ip_hmac_v1:safe-derived-value", UserAgent: "bounded-agent",
		}
		if err := writer.Append(context.Background(), event); err != nil {
			t.Fatalf("append %s: %v", action, err)
		}
	}
	if len(store.records) != len(actions) {
		t.Fatalf("record count=%d want=%d", len(store.records), len(actions))
	}
	if store.records[0].ActorUserID == nil || store.records[0].SessionID == nil || store.records[0].TokenFamilyID == nil {
		t.Fatal("safe identifiers were not mapped")
	}

	unknownActor := application.AuditEvent{
		OccurredAt: time.Now().UTC(), Action: application.AuditLoginFailure,
		Result: application.AuditResultFailure, Severity: application.AuditSeverityWarning, CorrelationID: "corr-unknown",
	}
	if err := writer.Append(context.Background(), unknownActor); err != nil {
		t.Fatal(err)
	}
	if store.records[len(store.records)-1].ActorUserID != nil {
		t.Fatal("unknown actor became a persisted identifier")
	}
}

func TestPostgresWriterRejectsUnsafeEventsAndRedactsStoreErrors(t *testing.T) {
	secret := "password-token-private-key"
	store := &captureStore{err: errors.New(secret)}
	writer := NewPostgresWriter(store)
	valid := application.AuditEvent{
		OccurredAt: time.Now().UTC(), Action: application.AuditLoginFailure,
		Result: application.AuditResultFailure, Severity: application.AuditSeverityWarning, CorrelationID: "corr-safe",
	}
	if err := writer.Append(context.Background(), valid); !errors.Is(err, application.ErrAuditPersistence) || strings.Contains(err.Error(), secret) {
		t.Fatalf("unsafe persistence error: %v", err)
	}
	invalid := valid
	invalid.NetworkIdentityHash = "192.0.2.1"
	if err := NewPostgresWriter(&captureStore{}).Append(context.Background(), invalid); !errors.Is(err, application.ErrAuditPersistence) {
		t.Fatalf("raw network identity accepted: %v", err)
	}
	invalid = valid
	invalid.UserAgent = strings.Repeat("x", MaximumUserAgentLength+1)
	if err := NewPostgresWriter(&captureStore{}).Append(context.Background(), invalid); !errors.Is(err, application.ErrAuditPersistence) {
		t.Fatalf("unbounded user-agent accepted: %v", err)
	}
	invalid = valid
	invalid.UserAgent = "unsafe\nagent"
	if err := NewPostgresWriter(&captureStore{}).Append(context.Background(), invalid); !errors.Is(err, application.ErrAuditPersistence) {
		t.Fatalf("control characters accepted in user-agent: %v", err)
	}

	recordType := reflect.TypeOf(platformdatabase.AuthenticationAuditRecord{})
	for _, forbidden := range []string{"Password", "PasswordHash", "AccessToken", "RefreshToken", "TokenDigest", "Cookie", "AuthorizationHeader", "PrivateKey", "Metadata"} {
		if _, exists := recordType.FieldByName(forbidden); exists {
			t.Fatalf("audit persistence boundary exposes forbidden field %s", forbidden)
		}
	}
}
