package httpserver

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

type ErrorDetail struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
}

type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

func errorHandler(ctx *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	code := "INTERNAL_ERROR"
	message := "The request could not be completed"

	var fiberError *fiber.Error
	if errors.As(err, &fiberError) {
		status = fiberError.Code
		code = "HTTP_ERROR"
		if status == fiber.StatusNotFound {
			code = "NOT_FOUND"
			message = "The requested resource was not found"
		}
	}

	return ctx.Status(status).JSON(ErrorEnvelope{Error: ErrorDetail{
		Code: code, Message: message, CorrelationID: correlationID(ctx),
	}})
}
