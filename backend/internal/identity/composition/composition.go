// Package composition is the identity module's composition root. Runtime HTTP
// mounting remains deliberately blocked until the HTTPS-attestation decision
// recorded by AUTH-BE-003 is approved.
package composition

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	identityaudit "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/audit"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/authhmac"
	identitydatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/database"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/network"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/password"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/ratelimit"
	identityruntime "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/runtime"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/token"
	platformdatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
)

type LookupFunc func(string) (string, bool)

type Config struct {
	PublicOrigin      string
	TrustedProxyCIDRs []string
}

func Build(ctx context.Context, pool *pgxpool.Pool, environment string, lookup LookupFunc) (*application.Service, Config, error) {
	if pool == nil || lookup == nil {
		return nil, Config{}, application.ErrInvalidSecurityConfig
	}
	publicOrigin, ok := lookup("AUTH_PUBLIC_ORIGIN")
	if !ok || publicOrigin == "" || publicOrigin != strings.TrimSpace(publicOrigin) {
		return nil, Config{}, application.ErrInvalidSecurityConfig
	}
	trustedRaw, _ := lookup("AUTH_TRUSTED_PROXY_CIDRS")
	trusted, err := network.ParseTrustedProxyCIDRs(trustedRaw)
	if err != nil {
		return nil, Config{}, err
	}
	if (environment == "staging" || environment == "production") && len(trusted) == 0 {
		return nil, Config{}, application.ErrInvalidSecurityConfig
	}
	resolver, err := network.NewResolver(trusted)
	if err != nil {
		return nil, Config{}, err
	}
	networkKey, rateKey, err := authhmac.Load(authhmac.LookupFunc(lookup))
	if err != nil {
		return nil, Config{}, err
	}
	networkHasher, err := network.NewIdentityHasher(networkKey)
	if err != nil {
		return nil, Config{}, err
	}
	rateKeyDeriver, err := ratelimit.NewKeyDeriver(rateKey)
	if err != nil {
		return nil, Config{}, err
	}
	keyRing, err := loadKeyRing(environment, lookup)
	if err != nil {
		return nil, Config{}, err
	}
	accessTokens, err := token.NewAccessTokenAdapter(keyRing, publicOrigin)
	if err != nil {
		return nil, Config{}, err
	}
	passwords := password.New()
	dummyHash, err := passwords.Hash(ctx, "authentication-dummy-password")
	if err != nil {
		return nil, Config{}, errors.Join(application.ErrInvalidSecurityConfig, errors.New("dummy credential hash initialization failed"))
	}
	audit := identityaudit.NewPostgresWriter(platformdatabase.NewAuthenticationAuditStore(pool))
	limiter := ratelimit.NewPostgresLimiter(platformdatabase.NewAuthenticationRateLimitStore(pool))
	service, err := application.NewService(application.ServiceDependencies{
		Users:      identitydatabase.NewPostgresUserRepository(pool),
		Transactor: identitydatabase.NewPostgresTransactor(pool),
		Passwords:  passwords, AccessTokens: accessTokens, RefreshTokens: token.NewRefreshTokenAdapter(),
		NetworkResolver: resolver, NetworkHasher: networkHasher, Audit: audit,
		RateLimiter: limiter, RateKeys: rateKeyDeriver, Clock: identityruntime.Clock{}, IDs: identityruntime.IDGenerator{}, DummyHash: dummyHash,
	})
	if err != nil {
		return nil, Config{}, err
	}
	return service, Config{PublicOrigin: publicOrigin, TrustedProxyCIDRs: trusted}, nil
}

func loadKeyRing(environment string, lookup LookupFunc) (*token.KeyRing, error) {
	if environment == "development" {
		path, ok := lookup("AUTH_JWT_LOCAL_KEY_RING_PATH")
		if !ok || path == "" {
			return nil, application.ErrInvalidSecurityConfig
		}
		return token.LoadLocalKeyRing(path)
	}
	if environment != "test" && environment != "staging" && environment != "production" {
		return nil, application.ErrInvalidSecurityConfig
	}
	active, activeOK := lookup("AUTH_JWT_ACTIVE_KID")
	privateKey, privateOK := lookup("AUTH_JWT_ACTIVE_PRIVATE_KEY_B64")
	verification, verificationOK := lookup("AUTH_JWT_VERIFICATION_KEYS_JSON")
	if !activeOK || !privateOK || !verificationOK {
		return nil, application.ErrInvalidSecurityConfig
	}
	return token.LoadEnvironmentKeyRing(active, privateKey, verification)
}
