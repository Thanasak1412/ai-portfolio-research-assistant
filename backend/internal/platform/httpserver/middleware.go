package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
)

const correlationHeader = "X-Correlation-ID"
const correlationLocal = "correlation_id"

var validCorrelationID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

func correlationMiddleware(ctx *fiber.Ctx) error {
	identifier := ctx.Get(correlationHeader)
	if !validCorrelationID.MatchString(identifier) {
		identifier = newCorrelationID()
	}
	ctx.Locals(correlationLocal, identifier)
	ctx.Set(correlationHeader, identifier)
	return ctx.Next()
}

func requestLoggingMiddleware(logger *slog.Logger) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		started := time.Now()
		err := ctx.Next()
		logger.Info("http request completed",
			"method", ctx.Method(),
			"path", ctx.Route().Path,
			"status", ctx.Response().StatusCode(),
			"latency_ms", time.Since(started).Milliseconds(),
			"correlation_id", correlationID(ctx),
		)
		return err
	}
}

func correlationID(ctx *fiber.Ctx) string {
	identifier, _ := ctx.Locals(correlationLocal).(string)
	return identifier
}

func newCorrelationID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "correlation-id-unavailable"
	}
	return hex.EncodeToString(buffer)
}
