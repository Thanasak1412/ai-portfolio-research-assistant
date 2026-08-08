// Package composition is the identity module's composition root.
package composition

import (
	"context"
	"errors"
	"net/netip"
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
	identityhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/transport/http"
	platformdatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
)

type LookupFunc func(string) (string, bool)

type Config struct {
	PublicOrigin           string
	TrustedProxyCIDRs      []string
	TrustedHTTPSProxyCIDRs []string
}

func Build(ctx context.Context, pool *pgxpool.Pool, environment string, lookup LookupFunc) (*application.Service, Config, error) {
	if pool == nil || lookup == nil {
		return nil, Config{}, application.ErrInvalidSecurityConfig
	}
	configuration, err := loadConfig(environment, lookup)
	if err != nil {
		return nil, Config{}, err
	}
	publicOrigin, trusted := configuration.PublicOrigin, configuration.TrustedProxyCIDRs
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
	return service, configuration, nil
}

// BuildHTTP composes the identity application service and its HTTP transport.
func BuildHTTP(ctx context.Context, pool *pgxpool.Pool, environment string, lookup LookupFunc) (*identityhttp.Handler, error) {
	service, configuration, err := Build(ctx, pool, environment, lookup)
	if err != nil {
		return nil, err
	}
	attestor, err := identityhttp.NewTrustedHTTPSAttestor(configuration.TrustedHTTPSProxyCIDRs)
	if err != nil {
		return nil, err
	}
	return identityhttp.NewHandler(service, configuration.PublicOrigin, attestor)
}

func loadConfig(environment string, lookup LookupFunc) (Config, error) {
	publicOrigin, ok := lookup("AUTH_PUBLIC_ORIGIN")
	if !ok || publicOrigin == "" || publicOrigin != strings.TrimSpace(publicOrigin) {
		return Config{}, application.ErrInvalidSecurityConfig
	}
	trustedRaw, _ := lookup("AUTH_TRUSTED_PROXY_CIDRS")
	trusted, err := network.ParseTrustedProxyCIDRs(trustedRaw)
	if err != nil {
		return Config{}, err
	}
	httpsRaw, _ := lookup("AUTH_TRUSTED_HTTPS_PROXY_CIDRS")
	httpsTrusted, err := parseTrustedHTTPSProxyCIDRs(httpsRaw)
	if err != nil {
		return Config{}, err
	}
	if environment == "staging" || environment == "production" {
		if len(trusted) == 0 {
			return Config{}, application.ErrInvalidSecurityConfig
		}
		if len(httpsTrusted) == 0 {
			return Config{}, fieldError("AUTH_TRUSTED_HTTPS_PROXY_CIDRS is required")
		}
	}
	return Config{PublicOrigin: publicOrigin, TrustedProxyCIDRs: trusted, TrustedHTTPSProxyCIDRs: httpsTrusted}, nil
}

func parseTrustedHTTPSProxyCIDRs(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(part))
		if strings.TrimSpace(part) == "" || err != nil || prefix.Bits() == 0 {
			return nil, fieldError("AUTH_TRUSTED_HTTPS_PROXY_CIDRS is invalid")
		}
		result = append(result, prefix.Masked().String())
	}
	return result, nil
}

func fieldError(field string) error {
	return errors.Join(application.ErrInvalidSecurityConfig, errors.New(field))
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
