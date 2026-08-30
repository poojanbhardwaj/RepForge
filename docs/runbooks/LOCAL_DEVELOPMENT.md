# Local development runbook

## Start

1. Confirm Docker Desktop is running with WSL2 Linux containers.
2. Run `corepack pnpm install --frozen-lockfile`.
3. Run `pnpm run doctor` and `pnpm run setup`.
4. Run `pnpm run dev`, or start individual workspace commands.

The root task runner locates a per-user Docker Desktop CLI under `%LOCALAPPDATA%` when Docker is missing from the inherited `PATH`.

Before starting infrastructure, tests, development processes, or smoke checks, the runner rejects hostnames, wildcard binds, and non-loopback PostgreSQL, Redis, or S3 endpoints. PostgreSQL DSNs are parsed by the pinned pgx parser: the effective primary and every multi-host/TLS fallback must be a literal loopback IP, and host/port query overrides, service files, and other unapproved connection options are rejected. Use literal `127.0.0.1` or `::1` addresses. The local platform intentionally has no remote-dependency escape hatch.

## Local identity

`.env.local` contains a generated synthetic bearer token and is ignored. Never copy it to staging or production. The seeded subject is `local-user`; rerunning the seed is idempotent.

Expo/Metro starts with `--host localhost`. While `EXPO_PUBLIC_AUTH_MODE=dev` or `EXPO_PUBLIC_DEV_AUTH_TOKEN` is present, `EXPO_PUBLIC_API_URL` must be an HTTP origin whose host is a literal IPv4 `127/8` address or IPv6 `::1`; `localhost`, LAN/wildcard/remote addresses, user-info, and malformed origins fail preflight and runtime initialization. Expo public variables are bundled into development client code. Do not put client secrets or session secrets in any `EXPO_PUBLIC_*` variable.

For OIDC, follow [the Auth0 local verification runbook](AUTH0_LOCAL_VERIFICATION.md). Remove the public development token, switch both API and mobile auth modes deliberately, and rebuild the native development client after native Auth0 configuration changes. OIDC permits HTTP only to literal loopback in a development build; all other API origins must be canonical HTTPS.

## Diagnostics

- `pnpm run doctor`
- `docker compose --env-file .env.local ps`
- `docker compose --env-file .env.local logs postgres redis objectstore`
- `pnpm run db:migrate`
- `pnpm run smoke`
- `pnpm run images:verify`
- `pnpm run secrets`

`/healthz` can remain healthy when PostgreSQL is unavailable; `/readyz` must return 503. Do not publish local ports or use production data.

`pnpm run smoke` uses only the synthetic local identity. It verifies profile and onboarding reads/writes, authoritative idempotency replay/conflict, exact-origin CORS, unauthenticated rejection, and build metadata; it is not live Auth0 evidence.

`pnpm run images:production-gate` is expected to fail while Garage is local-only and release
SBOM/signature/provenance evidence is incomplete. Do not bypass that failure for a release.

## Stop and clean

`pnpm run infra:down` stops containers and preserves named volumes. `pnpm run clean` removes only build/test outputs. No automatic command deletes infrastructure volumes; disposable integration databases are dropped by tests.
