# RepForge delivery plan

Status values: `planned`, `in progress`, `complete`, `blocked`.

## Bootstrap vertical slice

Status: **complete**

Acceptance criteria:

- Clean-checkout local setup for infrastructure, API, mobile/web development, and tests.
- Health/readiness/version endpoints and a tested authenticated `GET/PATCH /v1/me` flow.
- PostgreSQL migration, explicit ownership query, optimistic concurrency, generated OpenAPI client, and accessible mobile profile states.
- Development authentication proven impossible in staging/production.
- CI without production credentials and honest production-readiness/security evidence.

Dependencies: Docker/Compose, Go, Node/pnpm, PostgreSQL, Redis, Garage. Docker Desktop is available through its per-user installation path and is discovered automatically by `tools/repforge.mjs`.

Residual risks: remote GitHub Actions execution has not run because no commit or push was authorized; this initial repository has no Git `HEAD`, so local build evidence truthfully uses the `uncommitted` sentinel and there is no project history to inspect, although the real history command is verified against a disposable prior-commit credential; Garage exposes no analyzable package inventory and remains local-only; release SBOM/signature/provenance evidence is absent; the waiver-free production scan exposes `CVE-2026-14456` in PostgreSQL and Redis, while their exact local-only waivers must be removed by 2026-09-30. Real identity, production infrastructure, native releases, and production security evidence remain future work.

## Final-review remediation

Batch 1 implementation status: **complete for findings 1–4; scanner evidence refreshed**.

- Finding 1: the pinned pgx parser now restricts DSN keys, rejects redirecting URL/connection options, and validates the effective primary and every fallback host as a literal loopback IP. Configuration loading, pools, migrations, seed entry points, setup preflight, and disposable integration create/drop paths revalidate before mutation.
- Finding 2: the generated-secret exception now matches only normalized `.env.local` or `/repo/.env.local`; nested `.env.local` paths receive no exception while exact generated rule/value/line validation remains intact.
- Finding 3: Git inspection now positively verifies the work tree, `HEAD` commit or unborn symbolic `HEAD`, and reachable history. Only a verified zero-commit repository skips history scanning; all inspection errors fail closed.
- Finding 4: mobile runtime and project preflight permit active development credentials only for an HTTP origin at a literal IPv4 `127/8` or IPv6 `::1` host. Invalid schemes, remote/wildcard addresses, hostnames, user-info, non-origin paths, alternate numeric forms, and malformed values fail before token exposure.

Targeted Batch 1 evidence was refreshed on 2026-08-29 after Docker recovery. `corepack pnpm run secrets` exited 0 from the exact trusted project path after verifying the pinned Gitleaks manifest, generated-token canary, unrelated same-file credential, nested `.env.local` credential, disposable prior-history credential, four exact working-tree fixtures, and a positively verified unborn repository. The exact repository path, not a wildcard, is present in Git `safe.directory`.

Batch 2 implementation status: **complete for findings 5–8**.

- Finding 5: authenticated profile storage operations now receive a 10-second context deadline while the existing 5/15/15/60-second server socket deadlines remain unchanged. Deadline expiry returns one stable `504 profile_request_timeout` envelope with a safe message. A blocking-repository test covers `GET` and `PATCH`, verifies the deadline and cancellation, completes within a one-second guard, and proves exactly one response header write.
- Finding 6: profile updates now execute in a repeatable-read transaction and build the response from the SQL `UPDATE ... RETURNING` row plus consents read through the same transaction. Ownership, active-user, and expected-version SQL predicates are unchanged; no post-commit profile reread remains. A real-PostgreSQL test uses a post-commit transaction barrier, not sleeps, and proves both a later version-3 update and a later disable cannot replace or invalidate the already committed version-2 mutation response.
- Finding 7: the mobile form hydrates only while pristine, stores its optimistic version baseline independently from background query data, preserves dirty display-name drafts and their original conflict basis through refetch/conflict invalidation, and establishes a new clean baseline after a successful save. The component regression edits locally, receives two newer server versions, submits version 1, observes the existing safe conflict flow, and retains the draft.
- Finding 8: workspace discovery now uses `fileURLToPath(import.meta.url)`. The executable Node regression copies the real module beneath a temporary path containing spaces and Unicode, imports it in a child Node process, and verifies the decoded workspace path.

Targeted Batch 2 evidence on 2026-08-29: profile Go tests passed; the integration-tagged users package compiled; the barrier concurrency test passed both `later_update` and `later_disable`; the focused mobile suite passed 5/5 and mobile typecheck passed; `node --test tools/repforge.test.mjs` passed 12/12; `format:check` and generation-drift checks exited 0. The restricted sandbox initially denied Go's user cache and pinned-tool network lookup; reruns used the repository-local Go cache and approved execution without weakening any gate.

Final Batch 2 verification on 2026-08-29: `corepack pnpm run setup`, `verify`, `security`, and `smoke` all exited 0. `verify` included format/generation freshness, lint/typechecks, Go and JavaScript tests, 19/19 mobile tests, executable contracts, the complete real-container integration suite, command builds, the Next.js production build, and Expo Android export. `security` repeated the full pinned Gitleaks evidence, reported no `govulncheck`, `gosec`, or pnpm audit findings, inventoried 53 PostgreSQL and 22 Redis packages under exact local waivers, and continued to report Garage as missing coverage/local-only. The immutable `golang:1.26.7-bookworm` Linux race gate exited 0 with the backend source mounted read-only. `images:production-gate` exited 1 as expected and fail-closed: PostgreSQL and Redis each retain two unsuppressed `CVE-2026-14456` findings and lack release SBOM/provenance, while Garage remains local-only with no vulnerability inventory or release SBOM/provenance. Prompt 1 has not started.

Batch 3 implementation status: **complete for the two actionable P2 final-review findings**.

- Serialization abort finding: profile updates retain the authoritative `UPDATE ... RETURNING` row and same-transaction consent read. A typed `*pgconn.PgError` SQLSTATE `40001` receives at most one immediate retry; the unchanged expected version is then classified through the existing `ErrConflict` path, and an exhausted serialization retry also fails safely as that conflict rather than exposing database details or returning HTTP 500. A real-PostgreSQL handler integration test forces two same-version PATCH transactions to establish repeatable-read snapshots behind a synchronization barrier before either can update or commit. It deterministically proves three total attempts (two overlapping transactions plus one bounded retry), exactly one `200`, exactly one defined `409 version_conflict`, and one valid persisted winner at version 2 without sleeps.
- Unicode length finding: the mobile profile schema now counts Unicode code points after trimming with JavaScript string iteration, matching Go rune counting and PostgreSQL `char_length` for supplementary characters. The React Native UTF-16 `maxLength={80}` cap was removed, so valid input is not silently truncated. Tests cover 80 and 81 supplementary characters, mixed ASCII/supplementary boundaries, normal ASCII boundaries, and the absence of the native truncation prop. The API limit and generated contract remain 80 code points and were not changed.

Targeted Batch 3 evidence on 2026-08-29: the users handler selection passed; the focused real-PostgreSQL users suite passed ownership, authoritative-response, and overlapping same-version PATCH tests; the two focused mobile suites passed 9/9 and mobile typecheck exited 0. `corepack pnpm run format` and `format:check` exited 0. The first `generate:check` attempt was blocked by sandboxed access to the pinned SQL generator; the approved rerun exited 0 and proved no generated SQL or OpenAPI client drift.

Final Batch 3 verification on 2026-08-29: `corepack pnpm run verify`, `security`, and `smoke` all exited 0. `verify` included generation/format freshness, lint/typechecks, Go and JavaScript tests, 23/23 mobile tests, executable contracts, the complete real-container integration suite, command builds, the Next.js production build, and Expo Android export. `security` again passed every pinned secret canary/history/working-tree check, reported no `govulncheck`, `gosec`, or complete-graph pnpm audit findings, inventoried 53 PostgreSQL and 22 Redis packages under their exact local waivers, and continued to report Garage as missing coverage/local-only. Smoke passed health/readiness/version, authenticated GET/PATCH, unauthenticated 401, route logging, and build metadata with `version=0.1.0-bootstrap`, `commit=uncommitted`, and `builtAt=2026-08-29T14:38:05.370Z`. The immutable `golang:1.26.7-bookworm@sha256:e8c859f5632dcfde7b32d2012b4351728f6437930887c2f6a91ea242459e5514` Linux `go test -race ./...` gate exited 0 with the repository mounted read-only. The first sandboxed `verify` attempt could not spawn Docker; its approved rerun is the recorded passing gate. Prompt 1 has not started.

## Future milestones

1. `planned` — OIDC authentication and resumable adult onboarding.
2. `planned` — Exercise catalog, starter programs, and offline-first workout logging.
3. `planned` — Deterministic progression engine.
4. `planned` — Progress, readiness, scheduling, and retention loop.
5. `planned` — Bounded nutrition MVP.
6. `planned` — Sandbox subscriptions and server-side entitlements.
7. `planned` — Admin/support/privacy operations and full observability.
8. `planned` — Dedicated security hardening and penetration-test preparation.
9. `planned` — Beta release-candidate audit.

## Bootstrap evidence

Verified locally on 2026-08-29 using only synthetic data and loopback services:

- `pnpm install --frozen-lockfile` passed with all six workspace projects already up to date and lifecycle scripts denied except the explicit `esbuild` and `unrs-resolver` allowlist.
- `pnpm run doctor` passed with Git 2.52.0, Go 1.26.6 using the `go1.26.7` toolchain directive, Node 24.16.0, pnpm 11.19.0, Docker 29.7.2, Compose 5.4.0, Buildx 0.36.1, and Docker daemon 29.7.2. Node and pnpm satisfy the declared ranges; CI remains pinned to Node 24.20.0/pnpm 11.24.0.
- `pnpm run setup` passed after loopback preflight: digest-pinned PostgreSQL 18.6, Redis 8.2.9, and Garage 2.3.0 became healthy on `127.0.0.1`; migration `000001_users.sql` remained at version 1 and the synthetic seed completed.
- `pnpm run generate` refreshed the generated OpenAPI client after version-schema constraints changed. The final `pnpm run verify` exited 0 after generation freshness, OpenAPI lint, formatting, Go vet, TypeScript lint/typechecks, Go and JavaScript tests, executable real-router contract drift checks, real-container integration tests, Go command builds, the Next.js production build, and Expo Android export. The Go suite includes the exact production middleware chain proving `GET /v1/me` survives OTel route propagation and a real 404 logs `unmatched`; the Node suite includes exact-image waiver, zero-inventory, production-blocker, and immutable-scanner regressions.
- `pnpm run test:integration` passed the PostgreSQL migration round trip and owner update plus denied/not-found updates for another identity without a profile, a missing identity, a wrong provider, and a disabled user; each case proves the owner record remains unchanged. Redis and Garage adapter tests also passed.
- `pnpm audit --audit-level high --json` audited the complete installed graph—759 runtime, 416 development, 91 optional, 1,224 total dependencies—and reported zero info/low/moderate/high/critical advisories.
- `pnpm run security` exited 0: digest-pinned Gitleaks detected its generated token canary, an unrelated token beside an exact allowed fixture value, and a token retained only in a disposable committed repository's prior history through the real history command. Gitleaks has no file-wide or path-wide credential exclusion; the working-tree scan accepted only four validated exact synthetic fixture findings by rule and line and found nothing else. The project has no `HEAD`, so there is no project history to inspect. `govulncheck` and `gosec` found no vulnerabilities/findings; the full pnpm audit found no advisories; all Compose/scanner registry manifests matched their SHA-256 pins; and the local Trivy scan reported zero unsuppressed high/critical OS findings. PostgreSQL and Redis exposed 53 and 22 OS packages and received separate waivers selected only by their exact local digests. Garage returned no inventory and was reported as `MISSING COVERAGE`, not clean.
- `pnpm run images:production-gate` exited 1 as required without mounting either local waiver. It reported two unsuppressed `CVE-2026-14456` findings in each of PostgreSQL and Redis plus missing release SBOM/provenance evidence; Garage remained blocked because it is local-only, has no vulnerability inventory, and lacks release SBOM/provenance evidence.
- `pnpm run smoke` exited 0 for health, readiness, authenticated `GET/PATCH /v1/me`, unauthenticated `401`, matched route-template logs, and exact linker metadata validation. The built API returned `version=0.1.0-bootstrap`, `commit=uncommitted`, and UTC `builtAt=2026-08-29T08:22:01.787Z`; `uncommitted` is required because no `HEAD` exists and committing was forbidden.
- `go test -race ./...` passed in `golang:1.26.7-bookworm@sha256:e8c859f5632dcfde7b32d2012b4351728f6437930887c2f6a91ea242459e5514` with the source mounted read-only. The native Windows race command remains unusable with the installed 32-bit MinGW compiler; CI runs on Linux.
- The SHA-pinned GitHub Actions workflow contains no production credentials or deploy job, injects `github.sha` plus a UTC build timestamp, and mirrors setup, verify, race, security/image scanning, and smoke checks. It was inspected locally but not executed remotely because commit/push authorization was not granted.

Bootstrap completion is not production readiness. OIDC, onboarding, broader authorization, offline workout behavior, production infrastructure, release builds, DAST, independent testing, legal review, and operational drills remain explicitly incomplete.
