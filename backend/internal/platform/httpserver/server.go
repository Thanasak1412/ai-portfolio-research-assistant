package httpserver

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type ReadinessChecker interface {
	Ping(context.Context) error
}

// V1RouteRegistrar lets a module mount its public v1 transport routes without
// making the platform server depend on that module's implementation.
type V1RouteRegistrar interface {
	Mount(fiber.Router)
}

type Server struct {
	app    *fiber.App
	logger *slog.Logger
}

func New(logger *slog.Logger, readiness ReadinessChecker, registrars ...V1RouteRegistrar) *Server {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler,
	})

	app.Use(correlationMiddleware)
	app.Use(requestLoggingMiddleware(logger))
	app.Use(recover.New())

	api := app.Group("/api/v1")
	api.Get("/health/live", func(ctx *fiber.Ctx) error {
		return ctx.JSON(fiber.Map{"status": "alive"})
	})
	api.Get("/health/ready", func(ctx *fiber.Ctx) error {
		if err := readiness.Ping(ctx.UserContext()); err != nil {
			logger.Warn("readiness check failed", "dependency", "database", "correlation_id", correlationID(ctx))
			return ctx.Status(fiber.StatusServiceUnavailable).JSON(ErrorEnvelope{Error: ErrorDetail{
				Code: "NOT_READY", Message: "The service is not ready", CorrelationID: correlationID(ctx),
			}})
		}
		return ctx.JSON(fiber.Map{"status": "ready"})
	})
	for _, registrar := range registrars {
		if registrar != nil {
			registrar.Mount(api)
		}
	}

	return &Server{app: app, logger: logger}
}

func (s *Server) Listen(address string) error {
	return s.app.Listen(address)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("http server shutdown requested")
	return s.app.ShutdownWithContext(ctx)
}

func (s *Server) App() *fiber.App {
	return s.app
}
