# Architecture

## System shape

RepForge is a pnpm monorepo containing a Go modular monolith, a separate worker process, an Expo mobile app, a Next.js web app, and generated/shared packages. Modules own domain rules and persistence. Cross-module asynchronous work will use a transactional outbox when the first real asynchronous behavior is introduced.

The API uses standard `net/http`, explicit constructors, PostgreSQL through `pgx` and `sqlc`, structured `slog` logs, OpenTelemetry-compatible HTTP instrumentation, and REST JSON described by OpenAPI 3.1. Redis and object storage remain adapters until a domain requires them.

## Authentication and request flows

```text
Mobile OIDC app -> Auth0 Authorization Code + PKCE -> access token
  -> generated OpenAPI client -> Go API -> strict OIDC/JWKS validation

Browser -> stateless encrypted Auth0 v4 server session -> Next.js BFF
  -> server-only access token -> Go API

Go API -> scope middleware -> users/onboarding service
  -> identity ownership predicate + transaction -> PostgreSQL
```

The deterministic bearer adapter remains only for local/test and cannot validate in staging/production. OIDC discovery and JWKS must be HTTPS, same-origin, canonical, bounded JSON; only RS256 signing keys are accepted. Access-token signature, issuer, audience, authorized client, expiry, issued-at/not-before, subject, optional verified-email claim, and requested scopes are checked. Unknown or duplicate authorization inputs fail closed.

The web browser never receives the API access token: the Next.js BFF reads the encrypted, HttpOnly session, refreshes server-side, enforces exact `Origin`, same-origin Fetch Metadata, JSON, request/response size limits, and idempotency syntax, then forwards only the allowlisted request. Mobile permits OIDC API traffic over canonical HTTPS, with literal loopback HTTP allowed only in a development build.

`/healthz` checks only the process. `/readyz` checks PostgreSQL with a tight timeout because it is required for `/v1/me` and `/v1/onboarding`. Redis and object-store outages do not make this API unready.

## Data decisions

- UUIDv7 application-generated opaque IDs.
- UTC timestamps in storage and user-timezone conversion at presentation boundaries.
- Identity references are separate from profiles; every user-owned query includes the identity ownership predicate.
- Optimistic `version` fields protect offline-editable resources.
- First-login provisioning is serialized per provider/subject. Onboarding writes lock the owner row and store an owner-scoped request hash plus authoritative response snapshot for idempotent replay.
- Adult and safety evidence timestamps are append-only for a given acceptance; Terms and Privacy acceptances record the exact configured document version. A version change re-gates presentation and completion until reaccepted.
- Money will use integer minor units plus ISO currency codes. Measurements will use integer/decimal-safe representations.

## Production reference, not deployed

The documented target is AWS `ap-south-1`: WAF/ALB, ECS Fargate API/web/worker services, RDS PostgreSQL, managed Redis, S3/CloudFront, KMS and Secrets Manager, SES-compatible email, backups, restricted networking, and separate staging/production accounts and Terraform state. No Terraform resource exists during bootstrap.
