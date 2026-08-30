# RepForge

RepForge is an India-first adaptive strength-training and nutrition adherence product. This repository implements the bootstrap profile slice and Milestone 1 authentication/onboarding slice; it is not production-ready.

## Prerequisites

- Git 2.52+
- Go 1.26.7 (the Go toolchain directive can fetch the required patch release)
- Node.js 24 LTS; 24.20.0 is the recorded baseline
- Corepack and pnpm 11.24.0
- Docker Desktop with the WSL2 backend and Docker Compose

Docker Desktop licensing must be appropriate for your organization. The default synthetic, loopback-only workflow needs no hosted-service account. Live OIDC testing requires separately configured Auth0 native and regular-web applications; this repository does not create or modify them.

## First run

```powershell
corepack pnpm install --frozen-lockfile
pnpm run doctor
pnpm run setup
pnpm run smoke
```

`setup` creates an ignored `.env.local` containing synthetic local credentials and a generated web-session key, verifies that the HTTP bind and PostgreSQL, Redis, and S3 URLs use literal loopback IPs, starts PostgreSQL, Redis, and Garage on loopback interfaces, applies migrations, and inserts one synthetic development profile. Auth0 client placeholders remain intentionally unusable until supplied manually. There is no remote-dependency or wildcard-bind escape hatch.

Start all development processes with:

```powershell
pnpm run dev
```

- API: `http://127.0.0.1:8080`
- Web: `http://127.0.0.1:3000`
- Expo/Metro: `http://127.0.0.1:8081`
- PostgreSQL: `127.0.0.1:5432`
- Redis: `127.0.0.1:6379`
- Local S3-compatible endpoint: `http://127.0.0.1:3900`

See [the local runbook](docs/runbooks/LOCAL_DEVELOPMENT.md) for individual services and troubleshooting.

Expo starts with `--host localhost`; the synthetic bearer token must not be served over LAN. The optional OIDC development-client workflow and required manual callback/logout configuration are documented in [the Auth0 verification runbook](docs/runbooks/AUTH0_LOCAL_VERIFICATION.md).

## Verification

```powershell
pnpm run format:check
pnpm run generate:check
pnpm run lint
pnpm run typecheck
pnpm run test
pnpm run test:integration
pnpm run build
pnpm run images:verify
pnpm run secrets
pnpm run security
pnpm run verify
```

`pnpm run images:production-gate` is intentionally blocked by the unsuppressed
`CVE-2026-14456` findings in PostgreSQL and Redis, Garage's missing vulnerability inventory and
local-only status, and absent verified release SBOM and publisher-provenance evidence. Garage is a
local fixture and must never be promoted. Production scans never mount the loopback-only local CVE
waivers.

`pnpm run clean` removes only application build/test outputs; it does not remove source or Docker volumes. `pnpm run infra:down` stops local services without deleting data.

## Current boundaries

The deterministic bearer identity remains available only in local/test. Real app paths support Auth0 Authorization Code + PKCE on mobile and an Auth0 v4 server session/BFF on web; the API validates issuer, audience, client, signature, time claims, and scopes. Onboarding is resumable and version-gates adult, legal-consent, preference, and safety completion. Live tenant/device verification is still manual and unperformed in this run. Workouts, progression, nutrition, billing, admin operations, Terraform, deployment, native store releases, and production security evidence remain intentionally incomplete; see [production readiness](docs/PRODUCTION_READINESS.md).
