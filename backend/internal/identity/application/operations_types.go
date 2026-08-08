package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

const (
	AccessTokenLifetime  = 15 * time.Minute
	SessionIdleLifetime  = 30 * 24 * time.Hour
	SessionAbsoluteLimit = 90 * 24 * time.Hour
)

var (
	ErrInvalidRequest          = errors.New("invalid authentication request")
	ErrRegistrationRejected    = errors.New("registration rejected")
	ErrAuthenticationFailed    = errors.New("authentication failed")
	ErrSessionRefreshRejected  = errors.New("session refresh rejected")
	ErrAccessTokenInvalid      = errors.New("access token invalid")
	ErrBrowserSecurityRejected = errors.New("browser security rejected")
	ErrAuthenticationService   = errors.New("authentication service unavailable")
)

type RateLimitError struct{ retryAfter time.Duration }

func NewRateLimitError(retryAfter time.Duration) error {
	if retryAfter <= 0 {
		retryAfter = time.Second
	}
	return &RateLimitError{retryAfter: retryAfter}
}

func (err *RateLimitError) Error() string             { return "authentication rate limit exceeded" }
func (err *RateLimitError) RetryAfter() time.Duration { return err.retryAfter }

type Clock interface{ Now() time.Time }

type IDGenerator interface {
	UserID() (domain.UserID, error)
	SessionID() (domain.SessionID, error)
	TokenFamilyID() (domain.TokenFamilyID, error)
}

type RequestMetadata struct {
	CorrelationID string
	DirectPeerIP  string
	ForwardedFor  string
	UserAgent     string
}

type CredentialsInput struct {
	Email    string
	Password string
	Metadata RequestMetadata
}

type RefreshInput struct {
	RawToken string
	Metadata RequestMetadata
}

type SafeUser struct {
	ID        string
	Email     string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func safeUser(user domain.User) SafeUser {
	return SafeUser{ID: user.ID().String(), Email: user.Email().String(), Status: string(user.Status()), CreatedAt: user.CreatedAt(), UpdatedAt: user.UpdatedAt()}
}

type SessionResult struct {
	AccessToken     AccessToken
	User            SafeUser
	RefreshToken    RefreshToken
	CookieExpiresAt time.Time
	CookieMaxAge    time.Duration
}

func (SessionResult) MarshalJSON() ([]byte, error) { return nil, ErrRefreshTokenRejected }

type AccessResult struct{ AccessToken AccessToken }

type Operations interface {
	Register(context.Context, CredentialsInput) (SessionResult, error)
	Login(context.Context, CredentialsInput) (SessionResult, error)
	Refresh(context.Context, RefreshInput) (SessionResult, error)
	Logout(context.Context, RefreshInput) error
	CurrentUser(context.Context, domain.Principal) (SafeUser, error)
	ResolvePrincipal(context.Context, AccessToken) (domain.Principal, error)
}

var _ json.Marshaler = SessionResult{}
