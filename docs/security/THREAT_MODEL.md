# Bootstrap threat model

## Assets and boundaries

Assets are identity mappings, profile/consent records, local credentials, contract integrity, migration integrity, and source/dependency provenance. Boundaries exist at the mobile/API network edge, authentication middleware, domain/repository boundary, PostgreSQL, local Redis/S3 services, developer workstation, package registries, containers, and CI.

## Priority threats and controls

| Threat                      | Bootstrap control                                                                  | Residual work                                            |
| --------------------------- | ---------------------------------------------------------------------------------- | -------------------------------------------------------- |
| Broken object authorization | `/v1/me` derives ownership from authenticated subject; negative tests              | Full role/object matrix in later modules                 |
| Test-auth escape            | Environment/auth-mode validation before listener; fixed subject/token              | Replace application paths with OIDC + PKCE               |
| Token/sensitive log leakage | Allowlisted structured fields; no headers/bodies/profile values; redaction tests   | Central access/retention/alert controls                  |
| Injection/mass assignment   | `sqlc` parameterization, explicit patch DTO, unknown-field rejection               | Repeat for every endpoint                                |
| Account enumeration         | Same safe authentication error; no arbitrary user lookup                           | Rate limits/abuse monitoring with real identity          |
| Offline replay              | No bootstrap offline writes; optimistic version on profile                         | Idempotency ledger and ordered workout queue             |
| Admin abuse                 | No admin surface exists                                                            | Default-deny RBAC, reasons, audit events                 |
| Supply-chain compromise     | Exact direct pins, lockfiles, restricted install scripts, pinned CI actions, scans | Release provenance/SBOM and patch SLA ownership          |
| Local service exposure      | Loopback bindings and synthetic credentials                                        | Private production networking and managed secrets        |
| Backup/deletion failure     | No production backups/data                                                         | Implement and test resumable deletion and restore drills |
| Unsafe recommendations      | No recommendations exist                                                           | Deterministic versioned rules and pain boundaries        |

Threat model review is required whenever authentication, billing, data sharing, admin, AI, file handling, or infrastructure changes.
