# ASVS/MASVS evidence register

This is a bootstrap evidence index, not a compliance claim.

| Area                           | Target                      | Bootstrap evidence                                  | Status                                |
| ------------------------------ | --------------------------- | --------------------------------------------------- | ------------------------------------- |
| ASVS architecture/threat model | L2 + selected L3            | Architecture, ADRs, threat model                    | Partial                               |
| ASVS authentication            | L2                          | Fail-closed dev-auth config and tests               | Partial; real OIDC absent             |
| ASVS authorization             | L2/L3 selected              | Ownership predicates and post-commit barrier tests  | Partial                               |
| ASVS validation/API            | L2                          | Strict decoding, deadline cancellation, contracts   | Partial                               |
| ASVS logging                   | L2/L3 selected              | Matched-route structured logs and leakage tests     | Partial                               |
| ASVS data protection           | L2/L3 selected              | Classification/inventory; synthetic-only local data | Partial                               |
| MASVS AUTH/STORAGE/NETWORK     | Applicable release controls | Dev token not persisted; HTTP local only            | Blocked on native/OIDC release builds |
| MASVS PRIVACY/RESILIENCE       | Applicable release controls | Data minimization documentation                     | Blocked on native assessment          |

Every applicable ASVS 5.0 and MASVS/MASTG requirement must receive an evidence link or justified not-applicable decision before public launch.

## Dated bootstrap automation

On 2026-08-29, local unit, executable contract-route, deadline/cancellation/single-response, negative ownership, deterministic post-commit update/disable, dirty mobile draft/conflict, space/Unicode workspace-path, literal-loopback environment-isolation, OTel-preserved route logging, real-container integration, metadata-validating smoke, digest-pinned working-tree secret scanning with exact-line/unrelated/nested/history checks and no file-wide/path-wide credential exclusion, `govulncheck`, `gosec`, complete pnpm graph audit, digest/OS image checks, and Linux Go race checks passed. The project has no `HEAD`, so history is skipped only after positively verifying an unborn repository. The image evidence includes two exact-digest expiring local QUIC-only OpenSSL waivers and explicitly missing Garage inventory; the waiver-free production image gate mounts neither waiver and fails on the unwaived CVE, Garage's missing coverage/local-only status, and absent release SBOM/publisher-provenance evidence. This validates only the narrow synthetic `/v1/me` bootstrap surface. It is not an ASVS or MASVS conformance claim and does not replace staging, native-binary, operational, or independent assessment evidence.
