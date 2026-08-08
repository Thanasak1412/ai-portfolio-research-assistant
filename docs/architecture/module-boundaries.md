# Module Boundaries

The Go backend is one modular monolith with separately runnable API and worker processes. Future domain modules live under `backend/internal/<module>` and may expose public domain/application packages. Another module must not import that module's infrastructure or transport implementation, write its tables, or depend on private types.

`backend/internal/platform` contains technical capabilities and cannot become a business-logic catch-all. New platform packages need one named responsibility. Generic `common`, `helpers`, `misc`, and `utils` packages are prohibited.

Cross-module communication will use explicit application/domain interfaces and internal events. Kafka, RabbitMQ, and distributed service boundaries are not part of MVP bootstrap.

Run `scripts/check-module-boundaries.sh` as modules are introduced. Architectural review remains necessary because a static check cannot detect direct table ownership violations.

Within Identity, `domain` imports no outer layer, `application` imports no Fiber or persistence implementation, and `transport` calls only the application/domain and public platform HTTP boundary. `identity/composition` is the only package permitted to assemble Identity infrastructure adapters into application services. Fiber handlers own DTO/cookie/header concerns and never import Identity infrastructure or sqlc packages.
