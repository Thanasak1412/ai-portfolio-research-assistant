package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

func TestRegisterCreatesUserSessionAndAuditInOneTransaction(t *testing.T) {
	fixture := newServiceFixture(t)
	result, err := fixture.service.Register(context.Background(), CredentialsInput{Email: " Person@Example.COM ", Password: "correct horse battery staple", Metadata: fixture.metadata})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.Email != "person@example.com" || result.RefreshToken.IsZero() || result.CookieMaxAge != SessionIdleLifetime {
		t.Fatalf("result = %+v", result)
	}
	if len(fixture.store.users) != 1 || len(fixture.store.sessions) != 1 {
		t.Fatalf("users=%d sessions=%d", len(fixture.store.users), len(fixture.store.sessions))
	}
	if !containsAudit(fixture.store.audits, AuditRegistrationSuccess) {
		t.Fatal("registration success audit missing")
	}
}

func TestLoginCredentialAndDisabledFailuresArePubliclyGeneric(t *testing.T) {
	for _, test := range []struct {
		name     string
		seed     bool
		disabled bool
		password string
	}{
		{"unknown", false, false, "correct horse battery staple"},
		{"incorrect", true, false, "different valid password"},
		{"disabled", true, true, "correct horse battery staple"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newServiceFixture(t)
			if test.seed {
				fixture.seedUser(t, test.disabled)
			}
			_, err := fixture.service.Login(context.Background(), CredentialsInput{Email: "person@example.com", Password: test.password, Metadata: fixture.metadata})
			if !errors.Is(err, ErrAuthenticationFailed) {
				t.Fatalf("error = %v", err)
			}
			if !containsAudit(fixture.store.audits, AuditLoginFailure) {
				t.Fatal("login failure audit missing")
			}
			if test.disabled && !containsAudit(fixture.store.audits, AuditDisabledAccountRejection) {
				t.Fatal("disabled audit missing")
			}
		})
	}
}

func TestRefreshRotatesThenReplayCommitsFamilyRevocationBeforeRejecting(t *testing.T) {
	fixture := newServiceFixture(t)
	user := fixture.seedUser(t, false)
	originalToken := "rt_v1_original"
	original := fixture.seedSession(t, user, originalToken)

	result, err := fixture.service.Refresh(context.Background(), RefreshInput{RawToken: originalToken, Metadata: fixture.metadata})
	if err != nil {
		t.Fatal(err)
	}
	if result.RefreshToken.IsZero() || result.CookieExpiresAt.After(original.AbsoluteExpiresAt()) {
		t.Fatal("rotation extended or omitted credential")
	}
	family := fixture.store.family(original.FamilyID())
	if len(family) != 2 || countState(family, domain.RefreshSessionStateActive) != 1 || countState(family, domain.RefreshSessionStateReplaced) != 1 {
		t.Fatalf("family after rotation = %+v", family)
	}

	_, err = fixture.service.Refresh(context.Background(), RefreshInput{RawToken: originalToken, Metadata: fixture.metadata})
	if !errors.Is(err, ErrSessionRefreshRejected) {
		t.Fatalf("error = %v", err)
	}
	family = fixture.store.family(original.FamilyID())
	if countState(family, domain.RefreshSessionStateRevoked) != 2 {
		t.Fatalf("replay did not commit revocation: %+v", family)
	}
	if !containsAudit(fixture.store.audits, AuditRefreshTokenReuse) || !containsAudit(fixture.store.audits, AuditTokenFamilyRevocation) {
		t.Fatal("replay security audits missing")
	}
}

func TestSuccessfulLoginResetsOnlyEmailFailureKeyAndCreatesSession(t *testing.T) {
	fixture := newServiceFixture(t)
	fixture.seedUser(t, false)
	result, err := fixture.service.Login(context.Background(), CredentialsInput{Email: "person@example.com", Password: "correct horse battery staple", Metadata: fixture.metadata})
	if err != nil {
		t.Fatal(err)
	}
	if result.AccessToken.Value() == "" || fixture.limiter.resets != 1 || len(fixture.store.sessions) != 1 {
		t.Fatalf("result=%+v resets=%d sessions=%d", result.User, fixture.limiter.resets, len(fixture.store.sessions))
	}
	if !containsAudit(fixture.store.audits, AuditLoginSuccess) {
		t.Fatal("login audit missing")
	}
}

func TestRefreshRejectsExpiredAndDisabledSessionsWithoutIssuingReplacement(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		fixture := newServiceFixture(t)
		user := fixture.seedUser(t, false)
		raw := "rt_v1_expired"
		createdAt := fixture.clock.Now().Add(-31 * 24 * time.Hour)
		fixture.seedSessionAt(t, user, raw, createdAt, fixture.clock.Now().Add(-time.Second), fixture.clock.Now().Add(59*24*time.Hour))
		_, err := fixture.service.Refresh(context.Background(), RefreshInput{RawToken: raw, Metadata: fixture.metadata})
		if !errors.Is(err, ErrSessionRefreshRejected) {
			t.Fatalf("error=%v", err)
		}
		if len(fixture.store.sessions) != 1 {
			t.Fatal("expired session produced replacement")
		}
	})
	t.Run("disabled", func(t *testing.T) {
		fixture := newServiceFixture(t)
		user := fixture.seedUser(t, true)
		raw := "rt_v1_disabled"
		fixture.seedSession(t, user, raw)
		_, err := fixture.service.Refresh(context.Background(), RefreshInput{RawToken: raw, Metadata: fixture.metadata})
		if !errors.Is(err, ErrSessionRefreshRejected) {
			t.Fatalf("error=%v", err)
		}
		if !containsAudit(fixture.store.audits, AuditDisabledAccountRejection) {
			t.Fatal("disabled refresh audit missing")
		}
		if len(fixture.store.sessions) != 1 {
			t.Fatal("disabled session produced replacement")
		}
	})
}

func TestPrincipalResolutionRechecksCurrentAccountStatus(t *testing.T) {
	fixture := newServiceFixture(t)
	user := fixture.seedUser(t, true)
	token, _ := NewAccessToken("access-" + user.ID().String())
	if _, err := fixture.service.ResolvePrincipal(context.Background(), token); !errors.Is(err, ErrAccessTokenInvalid) {
		t.Fatalf("error=%v", err)
	}
	if _, err := fixture.service.CurrentUser(context.Background(), domain.Principal{}); !errors.Is(err, ErrAccessTokenInvalid) {
		t.Fatalf("error=%v", err)
	}
}

type serviceFixture struct {
	service  *Service
	store    *memoryStore
	limiter  *fakeLimiter
	clock    fixedClock
	metadata RequestMetadata
	ids      *fakeIDs
}

func newServiceFixture(t *testing.T) *serviceFixture {
	t.Helper()
	now := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	limiter := &fakeLimiter{}
	ids := &fakeIDs{}
	dummy, _ := domain.NewPasswordHash("dummy")
	fixture := &serviceFixture{store: store, limiter: limiter, clock: fixedClock{now}, metadata: RequestMetadata{CorrelationID: "corr-1", DirectPeerIP: "127.0.0.1", UserAgent: "test-agent"}, ids: ids}
	service, err := NewService(ServiceDependencies{Users: store, Transactor: store, Passwords: fakePasswords{}, AccessTokens: fakeAccessTokens{}, RefreshTokens: &fakeRefreshTokens{}, NetworkResolver: fakeNetworkResolver{}, NetworkHasher: fakeNetworkHasher{}, Audit: store, RateLimiter: limiter, RateKeys: fakeRateKeys{}, Clock: fixture.clock, IDs: ids, DummyHash: dummy})
	if err != nil {
		t.Fatal(err)
	}
	fixture.service = service
	return fixture
}

func (fixture *serviceFixture) seedUser(t *testing.T, disabled bool) domain.User {
	t.Helper()
	id, _ := fixture.ids.UserID()
	email, _ := domain.NormalizeEmail("person@example.com")
	hash, _ := domain.NewPasswordHash("correct horse battery staple")
	var user domain.User
	var err error
	if disabled {
		disabledAt := fixture.clock.Now()
		user, err = domain.NewUser(id, email, hash, domain.AccountStatusDisabled, fixture.clock.Now(), fixture.clock.Now(), &disabledAt)
	} else {
		user, err = domain.NewActiveUser(id, email, hash, fixture.clock.Now())
	}
	if err != nil {
		t.Fatal(err)
	}
	fixture.store.users[id.String()] = user
	fixture.store.emails[email.String()] = id.String()
	return user
}

func (fixture *serviceFixture) seedSession(t *testing.T, user domain.User, raw string) domain.RefreshSession {
	return fixture.seedSessionAt(t, user, raw, fixture.clock.Now(), fixture.clock.Now().Add(SessionIdleLifetime), fixture.clock.Now().Add(SessionAbsoluteLimit))
}

func (fixture *serviceFixture) seedSessionAt(t *testing.T, user domain.User, raw string, createdAt, idleExpiry, absoluteExpiry time.Time) domain.RefreshSession {
	t.Helper()
	sessionID, _ := fixture.ids.SessionID()
	familyID, _ := fixture.ids.TokenFamilyID()
	digest := digestFor(t, raw)
	session, err := domain.NewActiveRefreshSession(sessionID, familyID, user.ID(), digest, createdAt, idleExpiry, absoluteExpiry, "ip_hmac_v1:test", "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	fixture.store.sessions[session.ID().String()] = session
	fixture.store.digests[string(digest.Bytes())] = session.ID().String()
	return session
}

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

type fakeIDs struct{ counter int }

func (generator *fakeIDs) next() uuid.UUID {
	generator.counter++
	return uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012d", generator.counter))
}
func (generator *fakeIDs) UserID() (domain.UserID, error) { return domain.NewUserID(generator.next()) }
func (generator *fakeIDs) SessionID() (domain.SessionID, error) {
	return domain.NewSessionID(generator.next())
}
func (generator *fakeIDs) TokenFamilyID() (domain.TokenFamilyID, error) {
	return domain.NewTokenFamilyID(generator.next())
}

type fakePasswords struct{}

func (fakePasswords) Hash(_ context.Context, password string) (domain.PasswordHash, error) {
	if len(password) < 12 {
		return domain.PasswordHash{}, ErrCredentialRejected
	}
	return domain.NewPasswordHash(password)
}
func (fakePasswords) Verify(_ context.Context, password string, hash domain.PasswordHash) (PasswordVerification, error) {
	if password != hash.Encoded() {
		return PasswordVerification{}, ErrCredentialRejected
	}
	return PasswordVerification{Verified: true}, nil
}

type fakeAccessTokens struct{}

func (fakeAccessTokens) Issue(_ context.Context, userID domain.UserID, _ time.Time) (AccessToken, error) {
	return NewAccessToken("access-" + userID.String())
}
func (fakeAccessTokens) Verify(_ context.Context, token AccessToken, _ time.Time) (domain.Principal, error) {
	id, err := domain.ParseUserID(strings.TrimPrefix(token.Value(), "access-"))
	if err != nil {
		return domain.Principal{}, ErrAccessTokenRejected
	}
	return domain.NewPrincipal(id)
}

type fakeRefreshTokens struct{ counter int }

func (tokens *fakeRefreshTokens) Generate(context.Context) (RefreshToken, error) {
	tokens.counter++
	return NewRefreshToken(fmt.Sprintf("rt_v1_generated_%d", tokens.counter))
}
func (*fakeRefreshTokens) Parse(raw string) (RefreshToken, error) { return NewRefreshToken(raw) }
func (*fakeRefreshTokens) Digest(token RefreshToken) (domain.TokenDigest, error) {
	digest := sha256.Sum256([]byte(token.Value()))
	return domain.NewTokenDigest(digest[:])
}

type fakeNetworkResolver struct{}

func (fakeNetworkResolver) Resolve(ClientNetworkRequest) (netip.Addr, error) {
	return netip.MustParseAddr("127.0.0.1"), nil
}

type fakeNetworkHasher struct{}

func (fakeNetworkHasher) Hash(netip.Addr) (string, error) { return "ip_hmac_v1:test", nil }

type fakeRateKeys struct{}

func rateKey() RateLimitKey { key, _ := NewRateLimitKey(make([]byte, 32)); return key }
func (fakeRateKeys) LoginEmailFailure(domain.NormalizedEmail) (RateLimitKey, error) {
	return rateKey(), nil
}
func (fakeRateKeys) LoginIPAttempt(netip.Addr) (RateLimitKey, error)        { return rateKey(), nil }
func (fakeRateKeys) RegistrationIPAttempt(netip.Addr) (RateLimitKey, error) { return rateKey(), nil }
func (fakeRateKeys) RefreshFamilyAttempt(domain.TokenFamilyID) (RateLimitKey, error) {
	return rateKey(), nil
}

type fakeLimiter struct{ resets int }

func (*fakeLimiter) Check(context.Context, RateLimitPolicy, RateLimitKey, time.Time) (RateLimitResult, error) {
	return RateLimitResult{Allowed: true}, nil
}
func (limiter *fakeLimiter) ResetLoginEmailFailures(context.Context, RateLimitKey) error {
	limiter.resets++
	return nil
}
func (*fakeLimiter) CleanupExpired(context.Context, time.Time) (int64, error) { return 0, nil }

type memoryStore struct {
	mu       sync.Mutex
	users    map[string]domain.User
	emails   map[string]string
	sessions map[string]domain.RefreshSession
	digests  map[string]string
	audits   []AuditEvent
}

func newMemoryStore() *memoryStore {
	return &memoryStore{users: map[string]domain.User{}, emails: map[string]string{}, sessions: map[string]domain.RefreshSession{}, digests: map[string]string{}}
}
func (store *memoryStore) Create(_ context.Context, user domain.User) (domain.User, error) {
	if _, ok := store.emails[user.Email().String()]; ok {
		return domain.User{}, ErrDuplicateIdentity
	}
	store.users[user.ID().String()] = user
	store.emails[user.Email().String()] = user.ID().String()
	return user, nil
}
func (store *memoryStore) FindByNormalizedEmail(_ context.Context, email domain.NormalizedEmail) (domain.User, error) {
	id, ok := store.emails[email.String()]
	if !ok {
		return domain.User{}, ErrUserNotFound
	}
	return store.users[id], nil
}
func (store *memoryStore) FindByID(_ context.Context, id domain.UserID) (domain.User, error) {
	user, ok := store.users[id.String()]
	if !ok {
		return domain.User{}, ErrUserNotFound
	}
	return user, nil
}
func (store *memoryStore) CompareAndSwapPasswordHash(_ context.Context, id domain.UserID, expected, replacement domain.PasswordHash, _ time.Time) (domain.User, error) {
	user, ok := store.users[id.String()]
	if !ok || user.PasswordHash().Encoded() != expected.Encoded() {
		return domain.User{}, ErrPersistenceConflict
	}
	return user, nil
}
func (store *memoryStore) CreateInitial(_ context.Context, session domain.RefreshSession) (domain.RefreshSession, error) {
	store.sessions[session.ID().String()] = session
	store.digests[string(session.TokenDigest().Bytes())] = session.ID().String()
	return session, nil
}
func (store *memoryStore) FindByIDSession(_ context.Context, id domain.SessionID) (domain.RefreshSession, error) {
	session, ok := store.sessions[id.String()]
	if !ok {
		return domain.RefreshSession{}, ErrSessionNotFound
	}
	return session, nil
}
func (store *memoryStore) ListFamily(_ context.Context, familyID domain.TokenFamilyID) ([]domain.RefreshSession, error) {
	return store.family(familyID), nil
}
func (store *memoryStore) MarkExpired(context.Context, time.Time) ([]domain.SessionID, error) {
	return nil, nil
}
func (store *memoryStore) DeleteInactiveBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (store *memoryStore) LockByDigest(_ context.Context, digest domain.TokenDigest) (domain.RefreshSession, error) {
	id, ok := store.digests[string(digest.Bytes())]
	if !ok {
		return domain.RefreshSession{}, ErrSessionNotFound
	}
	return store.sessions[id], nil
}
func (store *memoryStore) MarkReplaced(_ context.Context, session domain.RefreshSession) (domain.RefreshSession, error) {
	store.sessions[session.ID().String()] = session
	return session, nil
}
func (store *memoryStore) InsertReplacement(ctx context.Context, session domain.RefreshSession) (domain.RefreshSession, error) {
	return store.CreateInitial(ctx, session)
}
func (store *memoryStore) RevokeFamily(_ context.Context, familyID domain.TokenFamilyID, at time.Time, reason string) ([]domain.RefreshSession, error) {
	var result []domain.RefreshSession
	for id, session := range store.sessions {
		if session.FamilyID() != familyID {
			continue
		}
		if !session.IsRevoked() {
			revoked, err := session.Revoke(at, reason)
			if err != nil {
				return nil, err
			}
			store.sessions[id] = revoked
			session = revoked
		}
		result = append(result, session)
	}
	return result, nil
}
func (store *memoryStore) Append(_ context.Context, event AuditEvent) error {
	store.audits = append(store.audits, event)
	return nil
}
func (store *memoryStore) WithinTransaction(ctx context.Context, operation func(context.Context, TransactionRepositories) error) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	return operation(ctx, store)
}
func (store *memoryStore) Users() UserRepository { return store }
func (store *memoryStore) RefreshSessions() RefreshSessionTransactionRepository {
	return memorySessionRepository{store}
}
func (store *memoryStore) Audit() AuditWriter { return store }
func (store *memoryStore) family(id domain.TokenFamilyID) []domain.RefreshSession {
	var result []domain.RefreshSession
	for _, session := range store.sessions {
		if session.FamilyID() == id {
			result = append(result, session)
		}
	}
	return result
}

// Go cannot overload FindByID for the two repository interfaces, so expose the
// session implementation through a narrow adapter in transaction repositories.
type memorySessionRepository struct{ *memoryStore }

func (repository memorySessionRepository) FindByID(ctx context.Context, id domain.SessionID) (domain.RefreshSession, error) {
	return repository.FindByIDSession(ctx, id)
}

func digestFor(t *testing.T, raw string) domain.TokenDigest {
	t.Helper()
	sum := sha256.Sum256([]byte(raw))
	digest, err := domain.NewTokenDigest(sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
func containsAudit(events []AuditEvent, action AuditAction) bool {
	for _, event := range events {
		if event.Action == action {
			return true
		}
	}
	return false
}
func countState(sessions []domain.RefreshSession, state domain.RefreshSessionState) int {
	count := 0
	for _, session := range sessions {
		if session.State() == state {
			count++
		}
	}
	return count
}
