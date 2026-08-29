# RepForge contributor rules

## Layout

- `backend/`: Go modular monolith, worker, migrations, and OpenAPI contract.
- `apps/mobile/`: Expo consumer app. `apps/web/`: Next.js marketing/operations shell.
- `packages/`: generated API client, shared configuration, and design tokens.
- `docs/`: product, architecture, security evidence, readiness, and runbooks.

Do not add an empty domain or infrastructure directory. Add it with its first useful file.

## Commands

Run from the repository root: `pnpm run help`, `doctor`, `setup`, `dev`, `generate`, `format`, `format:check`, `lint`, `typecheck`, `test`, `test:integration`, `build`, `security`, `smoke`, `verify`, and `clean`.

## Conventions and safety

- OpenAPI is the HTTP contract source; regenerate `packages/api-client` after changes.
- SQL and migrations are the persistence source; use parameterized queries and explicit ownership predicates.
- IDs exposed outside a process are UUIDv7. Persist timestamps in UTC.
- Never log credentials, authorization headers, request bodies, sensitive values, or free text.
- Only synthetic fixtures may be used locally or in CI. Never put real credentials in the repository.
- Development authentication must fail closed outside `local` and `test`.
- Do not commit, push, deploy, purchase resources, or use production data/credentials without explicit authorization.
- `clean` may remove only the allowlisted generated paths enforced by `tools/repforge.mjs`.

## Definition of done

Update contracts, generated clients, migrations, tests, documentation, `PLANS.md`, and readiness evidence together. Run relevant format, lint, type, unit, integration, build, and security checks; report failures and skipped checks honestly. A compiling skeleton is not production-ready.
