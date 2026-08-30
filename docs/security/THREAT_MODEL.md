# Threat model through Milestone 1

## Assets and boundaries

Assets are identity mappings, OIDC/session/refresh credentials, profile/onboarding/consent evidence, idempotency records, local credentials, contract/migration integrity, and source/dependency provenance. Boundaries exist at Auth0-hosted authorization, the native app and protected credential storage, browser/Next.js BFF, mobile/API network edge, OIDC discovery/JWKS, authentication/scope middleware, domain/repository boundary, PostgreSQL, local Redis/S3 services, developer workstation, registries, containers, and CI.

## Priority threats and controls

| Threat                             | Current control                                                                                                          | Residual work                                                  |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------- |
| Broken object authorization        | Profile/onboarding derive owner from provider+subject; SQL ownership predicates; disabled/cross-owner tests              | Full role/object matrix in later modules                       |
| Test-auth escape                   | `dev` mode valid only in local/test; mobile public dev token restricted to loopback development                          | Independent environment/deployment review                      |
| Token substitution/forgery         | RS256-only JWKS, strict issuer/audience/client/time/subject/scope validation, bounded same-origin discovery and rotation | Live tenant negative tests, revocation/abuse controls          |
| Browser token/CSRF exposure        | HttpOnly encrypted session; token stays in BFF; exact Origin, Fetch Metadata, JSON/idempotency and size checks           | Production CSP/rate limits/DAST and centralized session alerts |
| Native credential/cache leakage    | Auth0 protected credentials; cache partition/clear by subject; terminal-expiry cleanup                                   | Release-binary MASVS storage/dynamic assessment                |
| Idempotent/offline replay abuse    | Owner-scoped key+request hash, locked transaction, authoritative response snapshot, mismatch conflict                    | Retention cleanup and ordered workout replay                   |
| Consent-version bypass             | Server derives completion from current versions; append-only evidence; clients display required version                  | Legal approval, publication integrity, retention/deletion      |
| Token/sensitive log leakage        | Allowlisted structured fields; no headers/bodies/free text; analytics events contain names/source/step only              | Central access/retention/alert controls                        |
| Injection/mass assignment          | `sqlc` parameters, strict DTO/enum/range/unknown-field parsing, bounded JSON                                             | Repeat for every endpoint                                      |
| Account enumeration/credential use | Safe auth errors; no arbitrary lookup; verified-email claim cannot be false                                              | Auth0 attack protection, MFA policy, rate limits               |
| Admin abuse                        | No admin surface exists                                                                                                  | Default-deny RBAC, reasons, audit events                       |
| Supply-chain compromise            | Exact direct pins, lockfiles, restricted install scripts, pinned CI actions, scans                                       | Release provenance/SBOM and patch SLA ownership                |
| Local service exposure             | Loopback bindings and synthetic credentials                                                                              | Private production networking and managed secrets              |
| Backup/deletion failure            | No production backups/data                                                                                               | Export/deletion/retention and restore drills                   |
| Unsafe recommendations             | No recommendations exist                                                                                                 | Deterministic versioned rules and pain boundaries              |

Threat model review is required whenever authentication, billing, data sharing, admin, AI, file handling, or infrastructure changes.
