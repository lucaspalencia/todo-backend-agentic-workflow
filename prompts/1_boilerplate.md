Create a Go REST API project for a task management application.

Tech stack and requirements:
- Go with chi router for HTTP routing
- PostgreSQL as the database
- Structured logging using slog
- Database migrations using golang-migrate (file-based SQL migrations in a /migrations directory)
- Docker Compose for local development with a PostgreSQL container
- A separate Docker Compose configuration (or override) for running integration tests against an isolated PostgreSQL instance

Architecture: Follow Domain-Driven Design (DDD) with clear layer separation:
- domain: entities, value objects, and repository interfaces (no external dependencies)
- application: use cases / service layer orchestrating domain logic
- infrastructure: database implementations of repository interfaces, HTTP handlers, router setup, and external adapters
- Each layer should only depend inward (infrastructure → application → domain)

Project structure should reflect these DDD layers while following Go conventions. Include:
- A Makefile with targets for: run, test, migrate-up, migrate-down, docker-up, docker-down
- Environment configuration loaded from a .env file
- A health check endpoint (GET /health) that verifies database connectivity
- Proper graceful shutdown handling

Do not implement any business features yet — just the foundation with the DDD folder structure in place.
