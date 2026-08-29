# ADR 0002: OpenAPI-first REST and SQL-first persistence

Status: accepted.

OpenAPI 3.1 owns HTTP shapes; `openapi-typescript` generates client types and contract tests validate handlers. PostgreSQL SQL and migrations own persistence; `sqlc` generates query code used behind repositories. Hand-maintained duplicate request types and ORM-generated schema are rejected.
