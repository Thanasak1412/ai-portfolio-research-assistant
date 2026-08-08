package application

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

const (
	revocationLogout = "current_session_logout"
	revocationReplay = "refresh_token_reuse"
)

type ServiceDependencies struct {
	Users           UserRepository
	Transactor      Transactor
	Passwords       PasswordService
	AccessTokens    AccessTokenService
	RefreshTokens   RefreshTokenService
	NetworkResolver ClientNetworkResolver
	NetworkHasher   NetworkIdentityHasher
	Audit           AuditWriter
	RateLimiter     RateLimiter
	RateKeys        RateLimitKeyDeriver
	Clock           Clock
	IDs             IDGenerator
	DummyHash       domain.PasswordHash
}

type Service struct{ dependencies ServiceDependencies }

func NewService(dependencies ServiceDependencies) (*Service, error) {
	if dependencies.Users == nil || dependencies.Transactor == nil ||
		dependencies.Passwords == nil || dependencies.AccessTokens == nil || dependencies.RefreshTokens == nil ||
		dependencies.NetworkResolver == nil || dependencies.NetworkHasher == nil || dependencies.Audit == nil ||
		dependencies.RateLimiter == nil || dependencies.RateKeys == nil || dependencies.Clock == nil || dependencies.IDs == nil || dependencies.DummyHash.IsZero() {
		return nil, ErrAuthenticationService
	}
	return &Service{dependencies: dependencies}, nil
}

func (service *Service) Register(ctx context.Context, input CredentialsInput) (SessionResult, error) {
	now := service.dependencies.Clock.Now().UTC()
	email, network, networkHash, err := service.prepareRequest(input.Email, input.Metadata)
	if err != nil {
		return SessionResult{}, ErrInvalidRequest
	}
	key, err := service.dependencies.RateKeys.RegistrationIPAttempt(network)
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	if err := service.checkRateLimit(ctx, RateLimitRegistrationIPAttempt, key, now); err != nil {
		return SessionResult{}, err
	}
	passwordHash, err := service.dependencies.Passwords.Hash(ctx, input.Password)
	if err != nil {
		service.appendFailureAudit(ctx, AuditRegistrationFailure, nil, nil, nil, input.Metadata, networkHash, now)
		return SessionResult{}, ErrInvalidRequest
	}
	userID, sessionID, familyID, err := service.initialIDs()
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	refreshToken, digest, err := service.newRefreshCredential(ctx)
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	user, err := domain.NewActiveUser(userID, email, passwordHash, now)
	if err != nil {
		return SessionResult{}, ErrInvalidRequest
	}
	absoluteExpiry := now.Add(SessionAbsoluteLimit)
	session, err := domain.NewActiveRefreshSession(sessionID, familyID, userID, digest, now, now.Add(SessionIdleLifetime), absoluteExpiry, networkHash, safeUserAgent(input.Metadata.UserAgent))
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	accessToken, err := service.dependencies.AccessTokens.Issue(ctx, userID, now)
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	err = service.dependencies.Transactor.WithinTransaction(ctx, func(txctx context.Context, repositories TransactionRepositories) error {
		if _, createErr := repositories.Users().Create(txctx, user); createErr != nil {
			return createErr
		}
		if _, createErr := repositories.RefreshSessions().CreateInitial(txctx, session); createErr != nil {
			return createErr
		}
		return repositories.Audit().Append(txctx, auditEvent(AuditRegistrationSuccess, AuditResultSuccess, AuditSeverityInfo, &userID, &sessionID, &familyID, input.Metadata, networkHash, now))
	})
	if err != nil {
		service.appendFailureAudit(ctx, AuditRegistrationFailure, nil, nil, nil, input.Metadata, networkHash, now)
		if errors.Is(err, ErrDuplicateIdentity) {
			return SessionResult{}, ErrRegistrationRejected
		}
		return SessionResult{}, ErrAuthenticationService
	}
	return SessionResult{AccessToken: accessToken, User: safeUser(user), RefreshToken: refreshToken, CookieExpiresAt: session.IdleExpiresAt(), CookieMaxAge: SessionIdleLifetime}, nil
}

func (service *Service) Login(ctx context.Context, input CredentialsInput) (SessionResult, error) {
	now := service.dependencies.Clock.Now().UTC()
	email, network, networkHash, err := service.prepareRequest(input.Email, input.Metadata)
	if err != nil {
		return SessionResult{}, ErrInvalidRequest
	}
	ipKey, err := service.dependencies.RateKeys.LoginIPAttempt(network)
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	if err := service.checkRateLimit(ctx, RateLimitLoginIPAttempt, ipKey, now); err != nil {
		return SessionResult{}, err
	}
	emailKey, err := service.dependencies.RateKeys.LoginEmailFailure(email)
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	if err := service.checkRateLimit(ctx, RateLimitLoginEmailFailure, emailKey, now); err != nil {
		return SessionResult{}, err
	}

	user, findErr := service.dependencies.Users.FindByNormalizedEmail(ctx, email)
	stored := service.dependencies.DummyHash
	if findErr == nil {
		stored = user.PasswordHash()
	}
	verification, verifyErr := service.dependencies.Passwords.Verify(ctx, input.Password, stored)
	if findErr != nil || verifyErr != nil || !verification.Verified || !user.IsActive() {
		actor := (*domain.UserID)(nil)
		if findErr == nil {
			id := user.ID()
			actor = &id
		}
		service.appendFailureAudit(ctx, AuditLoginFailure, actor, nil, nil, input.Metadata, networkHash, now)
		if findErr == nil && user.IsDisabled() {
			service.appendFailureAudit(ctx, AuditDisabledAccountRejection, actor, nil, nil, input.Metadata, networkHash, now)
		}
		return SessionResult{}, ErrAuthenticationFailed
	}
	if err := service.dependencies.RateLimiter.ResetLoginEmailFailures(ctx, emailKey); err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	if verification.NeedsRehash {
		service.bestEffortRehash(ctx, input.Password, user, now)
	}
	return service.createLoginSession(ctx, user, input.Metadata, networkHash, now)
}

func (service *Service) createLoginSession(ctx context.Context, user domain.User, metadata RequestMetadata, networkHash string, now time.Time) (SessionResult, error) {
	sessionID, familyID, err := service.sessionIDs()
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	refreshToken, digest, err := service.newRefreshCredential(ctx)
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	absoluteExpiry := now.Add(SessionAbsoluteLimit)
	session, err := domain.NewActiveRefreshSession(sessionID, familyID, user.ID(), digest, now, now.Add(SessionIdleLifetime), absoluteExpiry, networkHash, safeUserAgent(metadata.UserAgent))
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	accessToken, err := service.dependencies.AccessTokens.Issue(ctx, user.ID(), now)
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	err = service.dependencies.Transactor.WithinTransaction(ctx, func(txctx context.Context, repositories TransactionRepositories) error {
		if _, createErr := repositories.RefreshSessions().CreateInitial(txctx, session); createErr != nil {
			return createErr
		}
		userID := user.ID()
		return repositories.Audit().Append(txctx, auditEvent(AuditLoginSuccess, AuditResultSuccess, AuditSeverityInfo, &userID, &sessionID, &familyID, metadata, networkHash, now))
	})
	if err != nil {
		return SessionResult{}, ErrAuthenticationService
	}
	return SessionResult{AccessToken: accessToken, User: safeUser(user), RefreshToken: refreshToken, CookieExpiresAt: session.IdleExpiresAt(), CookieMaxAge: SessionIdleLifetime}, nil
}

func (service *Service) Refresh(ctx context.Context, input RefreshInput) (SessionResult, error) {
	now := service.dependencies.Clock.Now().UTC()
	token, err := service.dependencies.RefreshTokens.Parse(input.RawToken)
	if err != nil {
		service.appendFailureAudit(ctx, AuditRefreshFailure, nil, nil, nil, input.Metadata, "", now)
		return SessionResult{}, ErrSessionRefreshRejected
	}
	digest, err := service.dependencies.RefreshTokens.Digest(token)
	if err != nil {
		return SessionResult{}, ErrSessionRefreshRejected
	}
	_, networkHash, err := service.resolveNetwork(input.Metadata)
	if err != nil {
		return SessionResult{}, ErrInvalidRequest
	}
	var result SessionResult
	var rejectAfterCommit bool
	err = service.dependencies.Transactor.WithinTransaction(ctx, func(txctx context.Context, repositories TransactionRepositories) error {
		session, lockErr := repositories.RefreshSessions().LockByDigest(txctx, digest)
		if lockErr != nil {
			return lockErr
		}
		if session.IsReplaced() || session.IsRevoked() {
			if _, revokeErr := repositories.RefreshSessions().RevokeFamily(txctx, session.FamilyID(), now, revocationReplay); revokeErr != nil {
				return revokeErr
			}
			userID, sessionID, familyID := session.UserID(), session.ID(), session.FamilyID()
			if auditErr := repositories.Audit().Append(txctx, auditEvent(AuditRefreshTokenReuse, AuditResultFailure, AuditSeverityHigh, &userID, &sessionID, &familyID, input.Metadata, networkHash, now)); auditErr != nil {
				return auditErr
			}
			if auditErr := repositories.Audit().Append(txctx, auditEvent(AuditTokenFamilyRevocation, AuditResultSuccess, AuditSeverityHigh, &userID, &sessionID, &familyID, input.Metadata, networkHash, now)); auditErr != nil {
				return auditErr
			}
			if auditErr := repositories.Audit().Append(txctx, auditEvent(AuditRefreshFailure, AuditResultFailure, AuditSeverityHigh, &userID, &sessionID, &familyID, input.Metadata, networkHash, now)); auditErr != nil {
				return auditErr
			}
			rejectAfterCommit = true
			return nil
		}
		if session.IsExpiredAt(now) {
			return ErrSessionRefreshRejected
		}
		user, userErr := repositories.Users().FindByID(txctx, session.UserID())
		if userErr != nil {
			return userErr
		}
		if !user.IsActive() {
			userID, sessionID, familyID := user.ID(), session.ID(), session.FamilyID()
			if auditErr := repositories.Audit().Append(txctx, auditEvent(AuditDisabledAccountRejection, AuditResultFailure, AuditSeverityWarning, &userID, &sessionID, &familyID, input.Metadata, networkHash, now)); auditErr != nil {
				return auditErr
			}
			if auditErr := repositories.Audit().Append(txctx, auditEvent(AuditRefreshFailure, AuditResultFailure, AuditSeverityWarning, &userID, &sessionID, &familyID, input.Metadata, networkHash, now)); auditErr != nil {
				return auditErr
			}
			rejectAfterCommit = true
			return nil
		}
		familyKey, keyErr := service.dependencies.RateKeys.RefreshFamilyAttempt(session.FamilyID())
		if keyErr != nil {
			return ErrAuthenticationService
		}
		if rateErr := service.checkRateLimit(txctx, RateLimitRefreshFamilyAttempt, familyKey, now); rateErr != nil {
			return rateErr
		}
		newToken, newDigest, credentialErr := service.newRefreshCredential(txctx)
		if credentialErr != nil {
			return ErrAuthenticationService
		}
		newSessionID, idErr := service.dependencies.IDs.SessionID()
		if idErr != nil {
			return ErrAuthenticationService
		}
		idleExpiry := minTime(now.Add(SessionIdleLifetime), session.AbsoluteExpiresAt())
		replaced, replacement, planErr := session.PlanReplacement(domain.ReplacementInput{SessionID: newSessionID, TokenDigest: newDigest, CreatedAt: now, IdleExpiresAt: idleExpiry, NetworkIdentityHash: networkHash, UserAgent: safeUserAgent(input.Metadata.UserAgent)})
		if planErr != nil {
			return ErrSessionRefreshRejected
		}
		accessToken, issueErr := service.dependencies.AccessTokens.Issue(txctx, user.ID(), now)
		if issueErr != nil {
			return ErrAuthenticationService
		}
		if _, replaceErr := repositories.RefreshSessions().MarkReplaced(txctx, replaced); replaceErr != nil {
			return replaceErr
		}
		if _, insertErr := repositories.RefreshSessions().InsertReplacement(txctx, replacement); insertErr != nil {
			return insertErr
		}
		userID, familyID := user.ID(), session.FamilyID()
		if auditErr := repositories.Audit().Append(txctx, auditEvent(AuditRefreshSuccess, AuditResultSuccess, AuditSeverityInfo, &userID, &newSessionID, &familyID, input.Metadata, networkHash, now)); auditErr != nil {
			return auditErr
		}
		result = SessionResult{AccessToken: accessToken, User: safeUser(user), RefreshToken: newToken, CookieExpiresAt: idleExpiry, CookieMaxAge: idleExpiry.Sub(now)}
		return nil
	})
	if rejectAfterCommit && err == nil {
		return SessionResult{}, ErrSessionRefreshRejected
	}
	if err != nil {
		service.appendFailureAudit(ctx, AuditRefreshFailure, nil, nil, nil, input.Metadata, networkHash, now)
		if errors.Is(err, ErrSessionRefreshRejected) || errors.Is(err, ErrSessionNotFound) {
			return SessionResult{}, ErrSessionRefreshRejected
		}
		var rateErr *RateLimitError
		if errors.As(err, &rateErr) {
			return SessionResult{}, rateErr
		}
		return SessionResult{}, ErrAuthenticationService
	}
	return result, nil
}

func (service *Service) Logout(ctx context.Context, input RefreshInput) error {
	now := service.dependencies.Clock.Now().UTC()
	token, err := service.dependencies.RefreshTokens.Parse(input.RawToken)
	if err != nil {
		return ErrSessionRefreshRejected
	}
	digest, err := service.dependencies.RefreshTokens.Digest(token)
	if err != nil {
		return ErrSessionRefreshRejected
	}
	_, networkHash, err := service.resolveNetwork(input.Metadata)
	if err != nil {
		return ErrInvalidRequest
	}
	err = service.dependencies.Transactor.WithinTransaction(ctx, func(txctx context.Context, repositories TransactionRepositories) error {
		session, lockErr := repositories.RefreshSessions().LockByDigest(txctx, digest)
		if lockErr != nil || !session.IsActiveAt(now) {
			return ErrSessionRefreshRejected
		}
		if _, revokeErr := repositories.RefreshSessions().RevokeFamily(txctx, session.FamilyID(), now, revocationLogout); revokeErr != nil {
			return revokeErr
		}
		userID, sessionID, familyID := session.UserID(), session.ID(), session.FamilyID()
		if auditErr := repositories.Audit().Append(txctx, auditEvent(AuditLogout, AuditResultSuccess, AuditSeverityInfo, &userID, &sessionID, &familyID, input.Metadata, networkHash, now)); auditErr != nil {
			return auditErr
		}
		return repositories.Audit().Append(txctx, auditEvent(AuditTokenFamilyRevocation, AuditResultSuccess, AuditSeverityInfo, &userID, &sessionID, &familyID, input.Metadata, networkHash, now))
	})
	if err != nil {
		if errors.Is(err, ErrSessionRefreshRejected) || errors.Is(err, ErrSessionNotFound) {
			return ErrSessionRefreshRejected
		}
		return ErrAuthenticationService
	}
	return nil
}

func (service *Service) ResolvePrincipal(ctx context.Context, token AccessToken) (domain.Principal, error) {
	principal, err := service.dependencies.AccessTokens.Verify(ctx, token, service.dependencies.Clock.Now().UTC())
	if err != nil {
		return domain.Principal{}, ErrAccessTokenInvalid
	}
	userID, authenticated := principal.UserID()
	if !authenticated {
		return domain.Principal{}, ErrAccessTokenInvalid
	}
	user, err := service.dependencies.Users.FindByID(ctx, userID)
	if err != nil || !user.IsActive() {
		return domain.Principal{}, ErrAccessTokenInvalid
	}
	return principal, nil
}

func (service *Service) CurrentUser(ctx context.Context, principal domain.Principal) (SafeUser, error) {
	if !principal.IsAuthenticated() {
		return SafeUser{}, ErrAccessTokenInvalid
	}
	userID, authenticated := principal.UserID()
	if !authenticated {
		return SafeUser{}, ErrAccessTokenInvalid
	}
	user, err := service.dependencies.Users.FindByID(ctx, userID)
	if err != nil || !user.IsActive() {
		return SafeUser{}, ErrAccessTokenInvalid
	}
	return safeUser(user), nil
}

func (service *Service) prepareRequest(rawEmail string, metadata RequestMetadata) (domain.NormalizedEmail, netip.Addr, string, error) {
	email, err := domain.NormalizeEmail(rawEmail)
	if err != nil || strings.TrimSpace(metadata.CorrelationID) == "" {
		return domain.NormalizedEmail{}, netip.Addr{}, "", ErrInvalidRequest
	}
	network, networkHash, err := service.resolveNetwork(metadata)
	return email, network, networkHash, err
}

func (service *Service) resolveNetwork(metadata RequestMetadata) (netip.Addr, string, error) {
	address, err := service.dependencies.NetworkResolver.Resolve(ClientNetworkRequest{DirectPeerIP: metadata.DirectPeerIP, ForwardedFor: metadata.ForwardedFor})
	if err != nil {
		return netip.Addr{}, "", err
	}
	hash, err := service.dependencies.NetworkHasher.Hash(address)
	return address, hash, err
}

func (service *Service) checkRateLimit(ctx context.Context, policy RateLimitPolicy, key RateLimitKey, now time.Time) error {
	result, err := service.dependencies.RateLimiter.Check(ctx, policy, key, now)
	if err != nil {
		return ErrAuthenticationService
	}
	if !result.Allowed {
		return NewRateLimitError(result.RetryAfter)
	}
	return nil
}

func (service *Service) initialIDs() (domain.UserID, domain.SessionID, domain.TokenFamilyID, error) {
	userID, err := service.dependencies.IDs.UserID()
	if err != nil {
		return domain.UserID{}, domain.SessionID{}, domain.TokenFamilyID{}, err
	}
	sessionID, err := service.dependencies.IDs.SessionID()
	if err != nil {
		return domain.UserID{}, domain.SessionID{}, domain.TokenFamilyID{}, err
	}
	familyID, err := service.dependencies.IDs.TokenFamilyID()
	return userID, sessionID, familyID, err
}

func (service *Service) sessionIDs() (domain.SessionID, domain.TokenFamilyID, error) {
	sessionID, err := service.dependencies.IDs.SessionID()
	if err != nil {
		return domain.SessionID{}, domain.TokenFamilyID{}, err
	}
	familyID, err := service.dependencies.IDs.TokenFamilyID()
	return sessionID, familyID, err
}

func (service *Service) newRefreshCredential(ctx context.Context) (RefreshToken, domain.TokenDigest, error) {
	token, err := service.dependencies.RefreshTokens.Generate(ctx)
	if err != nil {
		return RefreshToken{}, domain.TokenDigest{}, err
	}
	digest, err := service.dependencies.RefreshTokens.Digest(token)
	return token, digest, err
}

func (service *Service) bestEffortRehash(ctx context.Context, password string, user domain.User, now time.Time) {
	replacement, err := service.dependencies.Passwords.Hash(ctx, password)
	if err != nil {
		return
	}
	_, _ = service.dependencies.Users.CompareAndSwapPasswordHash(ctx, user.ID(), user.PasswordHash(), replacement, now)
}

func (service *Service) appendFailureAudit(ctx context.Context, action AuditAction, userID *domain.UserID, sessionID *domain.SessionID, familyID *domain.TokenFamilyID, metadata RequestMetadata, networkHash string, now time.Time) {
	_ = service.dependencies.Audit.Append(ctx, auditEvent(action, AuditResultFailure, AuditSeverityWarning, userID, sessionID, familyID, metadata, networkHash, now))
}

func auditEvent(action AuditAction, result AuditResult, severity AuditSeverity, userID *domain.UserID, sessionID *domain.SessionID, familyID *domain.TokenFamilyID, metadata RequestMetadata, networkHash string, now time.Time) AuditEvent {
	return AuditEvent{OccurredAt: now, Action: action, Result: result, Severity: severity, ActorUserID: userID, CorrelationID: metadata.CorrelationID, SessionID: sessionID, TokenFamilyID: familyID, NetworkIdentityHash: networkHash, UserAgent: safeUserAgent(metadata.UserAgent)}
}

func safeUserAgent(value string) string {
	if value == "" || len(value) > 512 || !utf8.ValidString(value) {
		return ""
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return ""
		}
	}
	return value
}

func minTime(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

var _ Operations = (*Service)(nil)
