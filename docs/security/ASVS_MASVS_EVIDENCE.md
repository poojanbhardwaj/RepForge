# ASVS/MASVS evidence register

This is a Milestone 1 evidence index, not a compliance claim.

| Area                           | Target                      | Bootstrap evidence                                    | Status                                |
| ------------------------------ | --------------------------- | ----------------------------------------------------- | ------------------------------------- |
| ASVS architecture/threat model | L2 + selected L3            | Architecture, ADRs, threat model                      | Partial                               |
| ASVS authentication            | L2                          | Fail-closed dev mode; OIDC/JWKS/session/BFF tests     | Partial; live tenant evidence pending |
| ASVS authorization             | L2/L3 selected              | Scopes, owner predicates, concurrency/disabled tests  | Partial                               |
| ASVS validation/API            | L2                          | Strict/bounded decoding, idempotency, contracts       | Partial                               |
| ASVS logging                   | L2/L3 selected              | Matched-route structured logs and leakage tests       | Partial                               |
| ASVS data protection           | L2/L3 selected              | Classification/inventory; synthetic-only local data   | Partial                               |
| MASVS AUTH/STORAGE/NETWORK     | Applicable release controls | Native PKCE/refresh path; cache isolation; HTTPS gate | Blocked on device/release assessment  |
| MASVS PRIVACY/RESILIENCE       | Applicable release controls | Data minimization documentation                       | Blocked on native assessment          |

Every applicable ASVS 5.0 and MASVS/MASTG requirement must receive an evidence link or justified not-applicable decision before public launch.

## Milestone 1 implementation evidence

Automated coverage includes strict OIDC configuration/discovery/JWKS and claims, same-`kid` rotation, malformed/oversized token inputs, missing/wrong scopes, duplicate authorization headers, exact CORS, BFF session failures and exact-origin/idempotency/body/upstream limits, server-only token forwarding and refreshed-cookie retention, mobile credential-origin and consent re-gating behavior, onboarding retry/cache/offline states, owner isolation, disabled users, first-login and same-key concurrency, authoritative replay/conflict, legal-version re-gating, migration round trip, and OpenAPI/generated-client drift. Live Auth0 tenant, emulator/device, refresh/logout, native protected-storage, and production cookie evidence remain manual blockers documented in the runbook.

## Dated bootstrap automation

On 2026-08-29, local unit, executable contract-route, deadline/cancellation/single-response, negative ownership, deterministic post-commit update/disable, dirty mobile draft/conflict, space/Unicode workspace-path, literal-loopback environment-isolation, OTel-preserved route logging, real-container integration, metadata-validating smoke, digest-pinned working-tree secret scanning with exact-line/unrelated/nested/history checks and no file-wide/path-wide credential exclusion, `govulncheck`, `gosec`, complete pnpm graph audit, digest/OS image checks, and Linux Go race checks passed. At that bootstrap checkpoint the repository was unborn and the scanner positively verified that state.

On 2026-08-30, Milestone 1 `security` and the final full verification gate passed against the reachable bootstrap commit plus the dirty current tree. The secret scan accepted only four exact generated local-fixture lines and found no unallowlisted working-tree or history secret. `govulncheck`, `gosec`, and the complete pnpm audit found no vulnerability, finding, or advisory. The final gate passed OIDC/JWKS negatives, scope/owner/concurrency/idempotency/legal-re-gating coverage, 12 tooling tests, 3 API-client tests, 21 web tests, 53 mobile tests, all Go packages, real PostgreSQL/Redis/object-store integration, and Go/Next/Expo builds. Image evidence retains two exact-digest expiring local QUIC-only OpenSSL waivers and explicitly missing Garage inventory; the waiver-free production image gate mounts neither waiver and remains blocked by the unwaived CVE, Garage's missing coverage/local-only status, and absent release SBOM/publisher-provenance evidence. This is not an ASVS or MASVS conformance claim and does not replace staging, native-binary, operational, or independent assessment evidence.
