//go:build integration

package database

import (
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	identityaudit "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/audit"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/authhmac"
	identitynetwork "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/network"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/ratelimit"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/token"
	platformdatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
)

func TestApplicationConcurrentRefreshAllowsOneRotationThenRevokesReplayedFamily(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	user := repositoryUserFixture(t, now)
	users := NewPostgresUserRepository(pool)
	if _, err := users.Create(ctx, user); err != nil {
		t.Fatal(err)
	}

	refreshTokens := token.NewRefreshTokenAdapter()
	originalToken, err := refreshTokens.Generate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := refreshTokens.Digest(originalToken)
	if err != nil {
		t.Fatal(err)
	}
	session, err := domain.NewActiveRefreshSession(mustDomainSessionID(t, uuid.New()), mustDomainFamilyID(t, uuid.New()), user.ID(), digest, now, now.Add(30*24*time.Hour), now.Add(90*24*time.Hour), "ip_hmac_v1:integration", "integration-browser")
	if err != nil {
		t.Fatal(err)
	}
	sessions := NewPostgresRefreshSessionRepository(pool)
	if _, err := sessions.CreateInitial(ctx, session); err != nil {
		t.Fatal(err)
	}

	networkEncoded := base64.StdEncoding.EncodeToString(bytesOf(1))
	rateEncoded := base64.StdEncoding.EncodeToString(bytesOf(2))
	networkKey, rateKey, err := authhmac.ParsePair(networkEncoded, rateEncoded)
	if err != nil {
		t.Fatal(err)
	}
	networkHasher, err := identitynetwork.NewIdentityHasher(networkKey)
	if err != nil {
		t.Fatal(err)
	}
	rateKeys, err := ratelimit.NewKeyDeriver(rateKey)
	if err != nil {
		t.Fatal(err)
	}
	resolver, _ := identitynetwork.NewResolver(nil)
	dummy, _ := domain.NewPasswordHash("dummy")
	service, err := application.NewService(application.ServiceDependencies{
		Users: users, Transactor: NewPostgresTransactor(pool), Passwords: integrationPasswords{},
		AccessTokens: integrationAccessTokens{}, RefreshTokens: refreshTokens, NetworkResolver: resolver, NetworkHasher: networkHasher,
		Audit:       identityaudit.NewPostgresWriter(platformdatabase.NewAuthenticationAuditStore(pool)),
		RateLimiter: ratelimit.NewPostgresLimiter(platformdatabase.NewAuthenticationRateLimitStore(pool)), RateKeys: rateKeys,
		Clock: integrationClock{now: now.Add(time.Minute)}, IDs: integrationIDs{}, DummyHash: dummy,
	})
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	for range 2 {
		go func() {
			defer wait.Done()
			<-start
			_, refreshErr := service.Refresh(ctx, application.RefreshInput{RawToken: originalToken.Value(), Metadata: application.RequestMetadata{CorrelationID: "concurrent-refresh-" + uuid.NewString(), DirectPeerIP: "127.0.0.1", UserAgent: "integration-browser"}})
			results <- refreshErr
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	successes, rejected := 0, 0
	for result := range results {
		switch {
		case result == nil:
			successes++
		case errors.Is(result, application.ErrSessionRefreshRejected):
			rejected++
		default:
			t.Fatalf("unexpected concurrent refresh result: %v", result)
		}
	}
	if successes != 1 || rejected != 1 {
		t.Fatalf("successes=%d rejected=%d", successes, rejected)
	}
	family, err := sessions.ListFamily(ctx, session.FamilyID())
	if err != nil {
		t.Fatal(err)
	}
	if len(family) != 2 {
		t.Fatalf("family generations=%d", len(family))
	}
	for _, generation := range family {
		if generation.State() != domain.RefreshSessionStateRevoked {
			t.Fatalf("generation %s state=%s", generation.ID(), generation.State())
		}
	}
}

type integrationPasswords struct{}

func (integrationPasswords) Hash(context.Context, string) (domain.PasswordHash, error) {
	return domain.NewPasswordHash("unused")
}
func (integrationPasswords) Verify(context.Context, string, domain.PasswordHash) (application.PasswordVerification, error) {
	return application.PasswordVerification{}, application.ErrCredentialRejected
}

type integrationAccessTokens struct{}

func (integrationAccessTokens) Issue(_ context.Context, userID domain.UserID, _ time.Time) (application.AccessToken, error) {
	return application.NewAccessToken("integration-access-" + userID.String())
}
func (integrationAccessTokens) Verify(context.Context, application.AccessToken, time.Time) (domain.Principal, error) {
	return domain.Principal{}, application.ErrAccessTokenRejected
}

type integrationClock struct{ now time.Time }

func (clock integrationClock) Now() time.Time { return clock.now }

type integrationIDs struct{}

func (integrationIDs) UserID() (domain.UserID, error)       { return domain.NewUserID(uuid.New()) }
func (integrationIDs) SessionID() (domain.SessionID, error) { return domain.NewSessionID(uuid.New()) }
func (integrationIDs) TokenFamilyID() (domain.TokenFamilyID, error) {
	return domain.NewTokenFamilyID(uuid.New())
}
func bytesOf(value byte) []byte {
	result := make([]byte, 32)
	for index := range result {
		result[index] = value
	}
	return result
}
