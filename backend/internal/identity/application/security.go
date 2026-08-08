package application

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

var (
	ErrCredentialRejected     = errors.New("credential verification failed")
	ErrInvalidSecurityConfig  = errors.New("invalid authentication security configuration")
	ErrAccessTokenRejected    = errors.New("access token verification failed")
	ErrClientIdentityRejected = errors.New("client network identity rejected")
	ErrAuditPersistence       = errors.New("authentication audit persistence failed")
	ErrRateLimitUnavailable   = errors.New("authentication rate limit unavailable")
	ErrInvalidRateLimitKey    = errors.New("invalid authentication rate limit key")
)

type PasswordVerification struct {
	Verified    bool
	NeedsRehash bool
}

type PasswordService interface {
	Hash(context.Context, string) (domain.PasswordHash, error)
	Verify(context.Context, string, domain.PasswordHash) (PasswordVerification, error)
}

type AccessToken struct{ value string }

func NewAccessToken(value string) (AccessToken, error) {
	if value == "" {
		return AccessToken{}, ErrAccessTokenRejected
	}
	return AccessToken{value: value}, nil
}

func (token AccessToken) Value() string    { return token.value }
func (token AccessToken) String() string   { return "[REDACTED]" }
func (token AccessToken) GoString() string { return "application.AccessToken{[REDACTED]}" }

type AccessTokenService interface {
	Issue(context.Context, domain.UserID, time.Time) (AccessToken, error)
	Verify(context.Context, AccessToken, time.Time) (domain.Principal, error)
}

type ClientNetworkRequest struct {
	DirectPeerIP string
	ForwardedFor string
}

type ClientNetworkResolver interface {
	Resolve(ClientNetworkRequest) (netip.Addr, error)
}

type AuditAction string
type AuditResult string
type AuditSeverity string

const (
	AuditRegistrationSuccess      AuditAction   = "registration_success"
	AuditRegistrationFailure      AuditAction   = "registration_failure"
	AuditLoginSuccess             AuditAction   = "login_success"
	AuditLoginFailure             AuditAction   = "login_failure"
	AuditRefreshSuccess           AuditAction   = "refresh_success"
	AuditRefreshFailure           AuditAction   = "refresh_failure"
	AuditLogout                   AuditAction   = "logout"
	AuditTokenFamilyRevocation    AuditAction   = "token_family_revocation"
	AuditRefreshTokenReuse        AuditAction   = "refresh_token_reuse"
	AuditDisabledAccountRejection AuditAction   = "disabled_account_rejection"
	AuditResultSuccess            AuditResult   = "success"
	AuditResultFailure            AuditResult   = "failure"
	AuditSeverityInfo             AuditSeverity = "info"
	AuditSeverityWarning          AuditSeverity = "warning"
	AuditSeverityHigh             AuditSeverity = "high"
)

type AuditEvent struct {
	OccurredAt          time.Time
	Action              AuditAction
	Result              AuditResult
	Severity            AuditSeverity
	ActorUserID         *domain.UserID
	CorrelationID       string
	SessionID           *domain.SessionID
	TokenFamilyID       *domain.TokenFamilyID
	NetworkIdentityHash string
	UserAgent           string
}

type AuditWriter interface {
	Append(context.Context, AuditEvent) error
}

type RateLimitPolicy string

const (
	RateLimitLoginEmailFailure     RateLimitPolicy = "login_email_failure"
	RateLimitLoginIPAttempt        RateLimitPolicy = "login_ip_attempt"
	RateLimitRegistrationIPAttempt RateLimitPolicy = "registration_ip_attempt"
	RateLimitRefreshFamilyAttempt  RateLimitPolicy = "refresh_family_attempt"
	RateLimitPolicyVersion                         = "v1"
	RateLimitKeyLength                             = 32
)

type RateLimitKey struct {
	value [RateLimitKeyLength]byte
	valid bool
}

func NewRateLimitKey(value []byte) (RateLimitKey, error) {
	if len(value) != RateLimitKeyLength {
		return RateLimitKey{}, ErrInvalidRateLimitKey
	}
	var key RateLimitKey
	copy(key.value[:], value)
	key.valid = true
	return key, nil
}

func (key RateLimitKey) Bytes() []byte {
	if !key.valid {
		return nil
	}
	result := make([]byte, len(key.value))
	copy(result, key.value[:])
	return result
}

func (key RateLimitKey) IsZero() bool     { return !key.valid }
func (key RateLimitKey) String() string   { return "[REDACTED]" }
func (key RateLimitKey) GoString() string { return "application.RateLimitKey{[REDACTED]}" }

type RateLimitResult struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}

type RateLimiter interface {
	Check(context.Context, RateLimitPolicy, RateLimitKey, time.Time) (RateLimitResult, error)
	ResetLoginEmailFailures(context.Context, RateLimitKey) error
	CleanupExpired(context.Context, time.Time) (int64, error)
}
