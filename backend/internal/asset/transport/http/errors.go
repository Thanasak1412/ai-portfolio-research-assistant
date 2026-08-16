package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/application"
	platformhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
)

var errInvalidRequest = errors.New("invalid asset HTTP request")

func writeError(ctx *fiber.Ctx, err error) error {
	status, code, message := fiber.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred"
	switch {
	case errors.Is(err, errInvalidRequest), errors.Is(err, application.ErrInvalidAssetInput):
		status, code, message = fiber.StatusBadRequest, "INVALID_REQUEST", "The request is invalid"
	case errors.Is(err, application.ErrUnauthenticated):
		status, code, message = fiber.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "The access token is invalid"
	case errors.Is(err, application.ErrAssetNotFound):
		status, code, message = fiber.StatusNotFound, "ASSET_NOT_FOUND", "The Asset was not found"
	}
	return ctx.Status(status).JSON(platformhttp.ErrorEnvelope{Error: platformhttp.ErrorDetail{Code: code, Message: message, CorrelationID: platformhttp.CorrelationID(ctx)}})
}
