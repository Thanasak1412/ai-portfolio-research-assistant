package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	platformhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
)

const (
	RefreshCookieName = "pra_rt_v1"
	RefreshCookiePath = "/api/v1/auth"
	RequestedWith     = "portfolio-web"
	accessExpiresIn   = 900
)

type HTTPSAttestor interface {
	OriginalRequestWasHTTPS(*fiber.Ctx) bool
}

type Handler struct {
	operations    application.Operations
	publicOrigin  string
	httpsAttestor HTTPSAttestor
}

func NewHandler(operations application.Operations, publicOrigin string, attestor HTTPSAttestor) (*Handler, error) {
	if operations == nil || publicOrigin == "" || attestor == nil {
		return nil, application.ErrAuthenticationService
	}
	return &Handler{operations: operations, publicOrigin: publicOrigin, httpsAttestor: attestor}, nil
}

func (handler *Handler) Mount(router fiber.Router) {
	auth := router.Group("/auth")
	auth.Post("/register", handler.register)
	auth.Post("/login", handler.login)
	auth.Post("/refresh", handler.browserSecurity, handler.refresh)
	auth.Post("/logout", handler.browserSecurity, handler.logout)
	auth.Get("/me", handler.authenticateBearer, handler.me)
}

// BearerMiddleware exposes the already-approved Identity authentication
// boundary for composition-root reuse by other module transports. It does not
// expose token parsing or verification details.
func (handler *Handler) BearerMiddleware() fiber.Handler { return handler.authenticateBearer }

// PrincipalExtractor exposes only the validated-principal context boundary for
// composition-root injection. A module transport must never inspect tokens.
func (handler *Handler) PrincipalExtractor() func(*fiber.Ctx) (domain.Principal, bool) {
	return PrincipalFromContext
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authenticatedUserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type accessTokenResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int    `json:"expiresIn"`
}

type authenticatedSessionResponse struct {
	accessTokenResponse
	User authenticatedUserResponse `json:"user"`
}

type errorEnvelope struct {
	Error errorDetail `json:"error"`
}
type errorDetail struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
}

func (handler *Handler) register(ctx *fiber.Ctx) error {
	request, err := decodeCredentials(ctx)
	if err != nil {
		return writeError(ctx, application.ErrInvalidRequest)
	}
	result, err := handler.operations.Register(ctx.UserContext(), application.CredentialsInput{Email: request.Email, Password: request.Password, Metadata: requestMetadata(ctx)})
	if err != nil {
		return writeError(ctx, err)
	}
	setRefreshCookie(ctx, result.RefreshToken, result.CookieExpiresAt, result.CookieMaxAge)
	return ctx.Status(fiber.StatusCreated).JSON(sessionResponse(result))
}

func (handler *Handler) login(ctx *fiber.Ctx) error {
	request, err := decodeCredentials(ctx)
	if err != nil {
		return writeError(ctx, application.ErrInvalidRequest)
	}
	result, err := handler.operations.Login(ctx.UserContext(), application.CredentialsInput{Email: request.Email, Password: request.Password, Metadata: requestMetadata(ctx)})
	if err != nil {
		return writeError(ctx, err)
	}
	setRefreshCookie(ctx, result.RefreshToken, result.CookieExpiresAt, result.CookieMaxAge)
	return ctx.JSON(sessionResponse(result))
}

func (handler *Handler) refresh(ctx *fiber.Ctx) error {
	raw := ctx.Cookies(RefreshCookieName)
	if raw == "" {
		clearRefreshCookie(ctx)
		return writeError(ctx, application.ErrSessionRefreshRejected)
	}
	result, err := handler.operations.Refresh(ctx.UserContext(), application.RefreshInput{RawToken: raw, Metadata: requestMetadata(ctx)})
	if err != nil {
		clearRefreshCookie(ctx)
		return writeError(ctx, err)
	}
	setRefreshCookie(ctx, result.RefreshToken, result.CookieExpiresAt, result.CookieMaxAge)
	return ctx.JSON(accessTokenResponse{AccessToken: result.AccessToken.Value(), TokenType: "Bearer", ExpiresIn: accessExpiresIn})
}

func (handler *Handler) logout(ctx *fiber.Ctx) error {
	raw := ctx.Cookies(RefreshCookieName)
	if raw == "" {
		clearRefreshCookie(ctx)
		return writeError(ctx, application.ErrSessionRefreshRejected)
	}
	err := handler.operations.Logout(ctx.UserContext(), application.RefreshInput{RawToken: raw, Metadata: requestMetadata(ctx)})
	clearRefreshCookie(ctx)
	if err != nil {
		return writeError(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (handler *Handler) me(ctx *fiber.Ctx) error {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return writeError(ctx, application.ErrAccessTokenInvalid)
	}
	user, err := handler.operations.CurrentUser(ctx.UserContext(), principal)
	if err != nil {
		return writeError(ctx, err)
	}
	return ctx.JSON(userResponse(user))
}

func (handler *Handler) browserSecurity(ctx *fiber.Ctx) error {
	if !handler.httpsAttestor.OriginalRequestWasHTTPS(ctx) || ctx.Get(fiber.HeaderOrigin) != handler.publicOrigin || ctx.Get("X-Requested-With") != RequestedWith {
		return writeError(ctx, application.ErrBrowserSecurityRejected)
	}
	return ctx.Next()
}

type principalContextKey struct{}

var validatedPrincipalKey = principalContextKey{}

func (handler *Handler) authenticateBearer(ctx *fiber.Ctx) error {
	raw := ctx.Get(fiber.HeaderAuthorization)
	parts := strings.Split(raw, " ")
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" || strings.ContainsAny(parts[1], "\t\r\n,") {
		return writeError(ctx, application.ErrAccessTokenInvalid)
	}
	token, err := application.NewAccessToken(parts[1])
	if err != nil {
		return writeError(ctx, application.ErrAccessTokenInvalid)
	}
	principal, err := handler.operations.ResolvePrincipal(ctx.UserContext(), token)
	if err != nil {
		return writeError(ctx, application.ErrAccessTokenInvalid)
	}
	ctx.Locals(validatedPrincipalKey, principal)
	return ctx.Next()
}

func PrincipalFromContext(ctx *fiber.Ctx) (domain.Principal, bool) {
	principal, ok := ctx.Locals(validatedPrincipalKey).(domain.Principal)
	return principal, ok && principal.IsAuthenticated()
}

func decodeCredentials(ctx *fiber.Ctx) (credentialsRequest, error) {
	mediaType, _, err := mime.ParseMediaType(ctx.Get(fiber.HeaderContentType))
	if err != nil || mediaType != fiber.MIMEApplicationJSON || len(ctx.Body()) == 0 {
		return credentialsRequest{}, application.ErrInvalidRequest
	}
	var request credentialsRequest
	decoder := json.NewDecoder(strings.NewReader(string(ctx.Body())))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return credentialsRequest{}, application.ErrInvalidRequest
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return credentialsRequest{}, application.ErrInvalidRequest
	}
	if request.Email == "" || request.Password == "" || !utf8.ValidString(request.Password) || utf8.RuneCountInString(request.Password) < 12 || len(request.Password) > 1024 {
		return credentialsRequest{}, application.ErrInvalidRequest
	}
	return request, nil
}

func requestMetadata(ctx *fiber.Ctx) application.RequestMetadata {
	direct := ctx.Context().RemoteAddr().String()
	host, _, err := net.SplitHostPort(direct)
	if err == nil {
		direct = host
	}
	return application.RequestMetadata{CorrelationID: platformhttp.CorrelationID(ctx), DirectPeerIP: direct, ForwardedFor: ctx.Get("X-Forwarded-For"), UserAgent: ctx.Get(fiber.HeaderUserAgent)}
}

func sessionResponse(result application.SessionResult) authenticatedSessionResponse {
	return authenticatedSessionResponse{accessTokenResponse: accessTokenResponse{AccessToken: result.AccessToken.Value(), TokenType: "Bearer", ExpiresIn: accessExpiresIn}, User: userResponse(result.User)}
}

func userResponse(user application.SafeUser) authenticatedUserResponse {
	return authenticatedUserResponse{ID: user.ID, Email: user.Email, Status: user.Status, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}
}

func setRefreshCookie(ctx *fiber.Ctx, token application.RefreshToken, expiresAt time.Time, lifetime time.Duration) {
	maxAge := int(math.Ceil(lifetime.Seconds()))
	maximum := int(application.SessionIdleLifetime.Seconds())
	if maxAge > maximum {
		maxAge = maximum
	}
	if maxAge < 1 {
		maxAge = 1
	}
	ctx.Cookie(&fiber.Cookie{Name: RefreshCookieName, Value: token.Value(), Path: RefreshCookiePath, MaxAge: maxAge, Expires: expiresAt.UTC(), Secure: true, HTTPOnly: true, SameSite: fiber.CookieSameSiteLaxMode})
}

func clearRefreshCookie(ctx *fiber.Ctx) {
	// Fiber omits Max-Age for zero and negative values, so use the explicit,
	// reviewed deletion representation required by AUTH_BROWSER_SECURITY-v1.
	ctx.Append(fiber.HeaderSetCookie, RefreshCookieName+"=; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:01 GMT; Path="+RefreshCookiePath+"; Secure; HttpOnly; SameSite=Lax")
}

func writeError(ctx *fiber.Ctx, err error) error {
	status, code, message := fiber.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred"
	switch {
	case errors.Is(err, application.ErrInvalidRequest):
		status, code, message = 400, "INVALID_REQUEST", "The request is invalid"
	case errors.Is(err, application.ErrRegistrationRejected):
		status, code, message = 409, "REGISTRATION_REJECTED", "Registration could not be completed"
	case errors.Is(err, application.ErrAuthenticationFailed):
		status, code, message = 401, "AUTHENTICATION_FAILED", "Authentication failed"
	case errors.Is(err, application.ErrSessionRefreshRejected):
		status, code, message = 401, "SESSION_REFRESH_REJECTED", "The Authentication session was rejected"
	case errors.Is(err, application.ErrAccessTokenInvalid):
		status, code, message = 401, "ACCESS_TOKEN_INVALID", "The access token is invalid"
	case errors.Is(err, application.ErrBrowserSecurityRejected):
		status, code, message = 403, "BROWSER_SECURITY_REJECTED", "The browser security requirements were not satisfied"
	case errors.Is(err, application.ErrAuthenticationService):
		status, code, message = 503, "AUTH_SERVICE_UNAVAILABLE", "Authentication is temporarily unavailable"
	default:
		var rateError *application.RateLimitError
		if errors.As(err, &rateError) {
			status, code, message = 429, "RATE_LIMIT_EXCEEDED", "Too many Authentication attempts"
			seconds := int64(math.Ceil(rateError.RetryAfter().Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			ctx.Set(fiber.HeaderRetryAfter, fmt.Sprintf("%d", seconds))
		}
	}
	return ctx.Status(status).JSON(errorEnvelope{Error: errorDetail{Code: code, Message: message, CorrelationID: platformhttp.CorrelationID(ctx)}})
}
