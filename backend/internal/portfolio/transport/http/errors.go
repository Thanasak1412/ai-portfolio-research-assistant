package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	platformhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/application"
)

func writeError(ctx *fiber.Ctx, err error) error {
	status, code, message := fiber.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred"
	switch {
	case errors.Is(err, application.ErrInvalidPortfolioInput), errors.Is(err, errInvalidJSON):
		status, code, message = fiber.StatusBadRequest, "INVALID_REQUEST", "The request is invalid"
	case errors.Is(err, application.ErrUnauthenticated):
		status, code, message = fiber.StatusUnauthorized, "ACCESS_TOKEN_INVALID", "The access token is invalid"
	case errors.Is(err, application.ErrPortfolioNotFound):
		status, code, message = fiber.StatusNotFound, "PORTFOLIO_NOT_FOUND", "The Portfolio was not found"
	case errors.Is(err, application.ErrPortfolioNameConflict):
		status, code, message = fiber.StatusConflict, "PORTFOLIO_NAME_CONFLICT", "The Portfolio name conflicts with an existing active Portfolio"
	case errors.Is(err, application.ErrPortfolioArchived):
		status, code, message = fiber.StatusUnprocessableEntity, "PORTFOLIO_ARCHIVED", "The Portfolio is archived"
	}
	return ctx.Status(status).JSON(platformhttp.ErrorEnvelope{Error: platformhttp.ErrorDetail{
		Code: code, Message: message, CorrelationID: platformhttp.CorrelationID(ctx),
	}})
}
