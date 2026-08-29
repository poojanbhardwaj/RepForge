# Architecture

## System shape

RepForge is a pnpm monorepo containing a Go modular monolith, a separate worker process, an Expo mobile app, a Next.js web app, and generated/shared packages. Modules own domain rules and persistence. Cross-module asynchronous work will use a transactional outbox when the first real asynchronous behavior is introduced.

The API uses standard `net/http`, explicit constructors, PostgreSQL through `pgx` and `sqlc`, structured `slog` logs, OpenTelemetry-compatible HTTP instrumentation, and REST JSON described by OpenAPI 3.1. Redis and object storage remain adapters until a domain requires them.

## Bootstrap request flow

```text
Expo profile screen
  -> generated OpenAPI fetch client
  -> HTTP limits / request ID / logging / recovery
  -> local-only bearer authenticator
  -> users service and ownership-scoped repository
  -> PostgreSQL
```

`/healthz` checks only the process. `/readyz` checks PostgreSQL with a tight timeout because it is the only dependency required to serve `/v1/me`. Redis and object-store outages do not make this bootstrap API unready.

## Data decisions

- UUIDv7 application-generated opaque IDs.
- UTC timestamps in storage and user-timezone conversion at presentation boundaries.
- Identity references are separate from profiles; every user-owned query includes the identity ownership predicate.
- Optimistic `version` fields protect offline-editable resources.
- Money will use integer minor units plus ISO currency codes. Measurements will use integer/decimal-safe representations.

## Production reference, not deployed

The documented target is AWS `ap-south-1`: WAF/ALB, ECS Fargate API/web/worker services, RDS PostgreSQL, managed Redis, S3/CloudFront, KMS and Secrets Manager, SES-compatible email, backups, restricted networking, and separate staging/production accounts and Terraform state. No Terraform resource exists during bootstrap.
