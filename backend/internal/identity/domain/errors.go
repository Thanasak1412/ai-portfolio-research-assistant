package domain

import "errors"

var (
	ErrInvalidUserID         = errors.New("invalid user identifier")
	ErrInvalidSessionID      = errors.New("invalid session identifier")
	ErrInvalidTokenFamilyID  = errors.New("invalid token family identifier")
	ErrInvalidEmail          = errors.New("invalid normalized email")
	ErrInvalidPasswordHash   = errors.New("invalid password credential")
	ErrInvalidTokenDigest    = errors.New("invalid token digest")
	ErrInvalidAccountStatus  = errors.New("invalid account status")
	ErrInvalidUser           = errors.New("invalid user")
	ErrUserDisabled          = errors.New("user account is disabled")
	ErrInvalidSessionState   = errors.New("invalid refresh session state")
	ErrInvalidRefreshSession = errors.New("invalid refresh session")
	ErrSessionInactive       = errors.New("refresh session is inactive")
	ErrSessionReplaced       = errors.New("refresh session has been replaced")
	ErrSessionRevoked        = errors.New("refresh session has been revoked")
	ErrSessionExpired        = errors.New("refresh session has expired")
	ErrSessionFamilyMismatch = errors.New("refresh session family mismatch")
	ErrInvalidReplacement    = errors.New("invalid refresh session replacement")
	ErrInvalidRevocation     = errors.New("invalid refresh session revocation")
	ErrUnauthenticated       = errors.New("authenticated principal is required")
)
