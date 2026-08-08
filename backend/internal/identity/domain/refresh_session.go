package domain

import (
	"fmt"
	"strings"
	"time"
)

type RefreshSessionState string

const (
	RefreshSessionStateActive   RefreshSessionState = "active"
	RefreshSessionStateReplaced RefreshSessionState = "replaced"
	RefreshSessionStateRevoked  RefreshSessionState = "revoked"
	RefreshSessionStateExpired  RefreshSessionState = "expired"
)

func ParseRefreshSessionState(value string) (RefreshSessionState, error) {
	state := RefreshSessionState(value)
	switch state {
	case RefreshSessionStateActive, RefreshSessionStateReplaced, RefreshSessionStateRevoked, RefreshSessionStateExpired:
		return state, nil
	default:
		return "", ErrInvalidSessionState
	}
}

type RefreshSessionData struct {
	ID                  SessionID
	FamilyID            TokenFamilyID
	UserID              UserID
	TokenDigest         TokenDigest
	State               RefreshSessionState
	ReplacementID       *SessionID
	CreatedAt           time.Time
	ReplacedAt          *time.Time
	IdleExpiresAt       time.Time
	AbsoluteExpiresAt   time.Time
	RevokedAt           *time.Time
	RevocationReason    string
	NetworkIdentityHash string
	UserAgent           string
}

type RefreshSession struct {
	id                  SessionID
	familyID            TokenFamilyID
	userID              UserID
	tokenDigest         TokenDigest
	state               RefreshSessionState
	replacementID       SessionID
	hasReplacementID    bool
	createdAt           time.Time
	replacedAt          time.Time
	hasReplacedAt       bool
	idleExpiresAt       time.Time
	absoluteExpiresAt   time.Time
	revokedAt           time.Time
	hasRevokedAt        bool
	revocationReason    string
	networkIdentityHash string
	userAgent           string
}

func NewRefreshSession(data RefreshSessionData) (RefreshSession, error) {
	if data.ID.IsZero() || data.FamilyID.IsZero() || data.UserID.IsZero() || data.TokenDigest.IsZero() {
		return RefreshSession{}, ErrInvalidRefreshSession
	}
	if _, err := ParseRefreshSessionState(string(data.State)); err != nil {
		return RefreshSession{}, err
	}
	if data.CreatedAt.IsZero() || !data.IdleExpiresAt.After(data.CreatedAt) || !data.AbsoluteExpiresAt.After(data.CreatedAt) || data.IdleExpiresAt.After(data.AbsoluteExpiresAt) {
		return RefreshSession{}, ErrInvalidRefreshSession
	}
	if len(data.NetworkIdentityHash) > 128 || len(data.UserAgent) > 512 {
		return RefreshSession{}, ErrInvalidRefreshSession
	}
	if data.NetworkIdentityHash != "" && strings.TrimSpace(data.NetworkIdentityHash) == "" {
		return RefreshSession{}, ErrInvalidRefreshSession
	}
	if data.UserAgent != "" && strings.TrimSpace(data.UserAgent) == "" {
		return RefreshSession{}, ErrInvalidRefreshSession
	}
	if err := validateLifecycle(data); err != nil {
		return RefreshSession{}, err
	}
	session := RefreshSession{
		id: data.ID, familyID: data.FamilyID, userID: data.UserID, tokenDigest: data.TokenDigest,
		state: data.State, createdAt: data.CreatedAt, idleExpiresAt: data.IdleExpiresAt,
		absoluteExpiresAt: data.AbsoluteExpiresAt, revocationReason: data.RevocationReason,
		networkIdentityHash: data.NetworkIdentityHash, userAgent: data.UserAgent,
	}
	if data.ReplacementID != nil {
		session.replacementID = *data.ReplacementID
		session.hasReplacementID = true
	}
	if data.ReplacedAt != nil {
		session.replacedAt = *data.ReplacedAt
		session.hasReplacedAt = true
	}
	if data.RevokedAt != nil {
		session.revokedAt = *data.RevokedAt
		session.hasRevokedAt = true
	}
	return session, nil
}

func NewActiveRefreshSession(
	id SessionID,
	familyID TokenFamilyID,
	userID UserID,
	digest TokenDigest,
	createdAt time.Time,
	idleExpiresAt time.Time,
	absoluteExpiresAt time.Time,
	networkIdentityHash string,
	userAgent string,
) (RefreshSession, error) {
	return NewRefreshSession(RefreshSessionData{
		ID: id, FamilyID: familyID, UserID: userID, TokenDigest: digest,
		State: RefreshSessionStateActive, CreatedAt: createdAt, IdleExpiresAt: idleExpiresAt,
		AbsoluteExpiresAt: absoluteExpiresAt, NetworkIdentityHash: networkIdentityHash, UserAgent: userAgent,
	})
}

func validateLifecycle(data RefreshSessionData) error {
	hasReplacement := data.ReplacementID != nil || data.ReplacedAt != nil
	if data.ReplacementID != nil && data.ReplacementID.IsZero() {
		return ErrInvalidReplacement
	}
	if data.ReplacementID != nil && *data.ReplacementID == data.ID {
		return ErrInvalidReplacement
	}
	if (data.ReplacementID == nil) != (data.ReplacedAt == nil) {
		return ErrInvalidRefreshSession
	}
	if data.ReplacedAt != nil && data.ReplacedAt.Before(data.CreatedAt) {
		return ErrInvalidRefreshSession
	}
	hasRevocation := data.RevokedAt != nil || data.RevocationReason != ""
	if (data.RevokedAt == nil) != (data.RevocationReason == "") {
		return ErrInvalidRevocation
	}
	if data.RevokedAt != nil && (data.RevokedAt.Before(data.CreatedAt) || strings.TrimSpace(data.RevocationReason) == "" || len(data.RevocationReason) > 128) {
		return ErrInvalidRevocation
	}
	switch data.State {
	case RefreshSessionStateActive, RefreshSessionStateExpired:
		if hasReplacement || hasRevocation {
			return ErrInvalidRefreshSession
		}
	case RefreshSessionStateReplaced:
		if !hasReplacement || hasRevocation {
			return ErrInvalidRefreshSession
		}
	case RefreshSessionStateRevoked:
		if !hasRevocation {
			return ErrInvalidRefreshSession
		}
	}
	return nil
}

type ReplacementInput struct {
	SessionID           SessionID
	TokenDigest         TokenDigest
	CreatedAt           time.Time
	IdleExpiresAt       time.Time
	NetworkIdentityHash string
	UserAgent           string
}

func (session RefreshSession) PlanReplacement(input ReplacementInput) (RefreshSession, RefreshSession, error) {
	if err := session.ReplacementEligibility(input.CreatedAt); err != nil {
		return RefreshSession{}, RefreshSession{}, err
	}
	if input.SessionID.IsZero() || input.SessionID == session.id {
		return RefreshSession{}, RefreshSession{}, ErrInvalidReplacement
	}
	replacement, err := NewActiveRefreshSession(
		input.SessionID, session.familyID, session.userID, input.TokenDigest, input.CreatedAt,
		input.IdleExpiresAt, session.absoluteExpiresAt, input.NetworkIdentityHash, input.UserAgent,
	)
	if err != nil {
		return RefreshSession{}, RefreshSession{}, fmt.Errorf("%w", ErrInvalidReplacement)
	}
	replacedAt := input.CreatedAt
	replacementID := input.SessionID
	replaced, err := NewRefreshSession(RefreshSessionData{
		ID: session.id, FamilyID: session.familyID, UserID: session.userID, TokenDigest: session.tokenDigest,
		State: RefreshSessionStateReplaced, ReplacementID: &replacementID, CreatedAt: session.createdAt,
		ReplacedAt: &replacedAt, IdleExpiresAt: session.idleExpiresAt, AbsoluteExpiresAt: session.absoluteExpiresAt,
		NetworkIdentityHash: session.networkIdentityHash, UserAgent: session.userAgent,
	})
	if err != nil {
		return RefreshSession{}, RefreshSession{}, err
	}
	return replaced, replacement, nil
}

func (session RefreshSession) ReplacementEligibility(now time.Time) error {
	switch session.state {
	case RefreshSessionStateReplaced:
		return ErrSessionReplaced
	case RefreshSessionStateRevoked:
		return ErrSessionRevoked
	case RefreshSessionStateExpired:
		return ErrSessionExpired
	case RefreshSessionStateActive:
		if session.IsExpiredAt(now) {
			return ErrSessionExpired
		}
		return nil
	default:
		return ErrSessionInactive
	}
}

func (session RefreshSession) Revoke(at time.Time, reason string) (RefreshSession, error) {
	if session.state == RefreshSessionStateRevoked {
		return RefreshSession{}, ErrSessionRevoked
	}
	if at.Before(session.createdAt) || strings.TrimSpace(reason) == "" || len(reason) > 128 {
		return RefreshSession{}, ErrInvalidRevocation
	}
	data := session.Data()
	data.State = RefreshSessionStateRevoked
	data.RevokedAt = &at
	data.RevocationReason = reason
	return NewRefreshSession(data)
}

func (session RefreshSession) Data() RefreshSessionData {
	data := RefreshSessionData{
		ID: session.id, FamilyID: session.familyID, UserID: session.userID, TokenDigest: session.tokenDigest,
		State: session.state, CreatedAt: session.createdAt, IdleExpiresAt: session.idleExpiresAt,
		AbsoluteExpiresAt: session.absoluteExpiresAt, RevocationReason: session.revocationReason,
		NetworkIdentityHash: session.networkIdentityHash, UserAgent: session.userAgent,
	}
	if session.hasReplacementID {
		value := session.replacementID
		data.ReplacementID = &value
	}
	if session.hasReplacedAt {
		value := session.replacedAt
		data.ReplacedAt = &value
	}
	if session.hasRevokedAt {
		value := session.revokedAt
		data.RevokedAt = &value
	}
	return data
}

func (session RefreshSession) ID() SessionID                { return session.id }
func (session RefreshSession) FamilyID() TokenFamilyID      { return session.familyID }
func (session RefreshSession) UserID() UserID               { return session.userID }
func (session RefreshSession) TokenDigest() TokenDigest     { return session.tokenDigest }
func (session RefreshSession) State() RefreshSessionState   { return session.state }
func (session RefreshSession) CreatedAt() time.Time         { return session.createdAt }
func (session RefreshSession) IdleExpiresAt() time.Time     { return session.idleExpiresAt }
func (session RefreshSession) AbsoluteExpiresAt() time.Time { return session.absoluteExpiresAt }
func (session RefreshSession) IsReplaced() bool             { return session.state == RefreshSessionStateReplaced }
func (session RefreshSession) IsRevoked() bool              { return session.state == RefreshSessionStateRevoked }
func (session RefreshSession) IsExpiredAt(now time.Time) bool {
	return session.state == RefreshSessionStateExpired || !now.Before(session.idleExpiresAt) || !now.Before(session.absoluteExpiresAt)
}
func (session RefreshSession) IsActiveAt(now time.Time) bool {
	return session.state == RefreshSessionStateActive && !session.IsExpiredAt(now)
}
func (session RefreshSession) String() string {
	return fmt.Sprintf("RefreshSession{id:%s,family:%s,state:%s}", session.id, session.familyID, session.state)
}
func (session RefreshSession) GoString() string { return session.String() }
