# RepForge product requirements through Milestone 1

## Product direction

RepForge helps adults in India follow safe strength training, log quickly, understand deterministic changes, and connect training with realistic nutrition. The product is for beginner-to-intermediate gym users and must support global expansion without becoming an all-sports application.

The repeated value loop is: receive an achievable workout, log it reliably including offline, give RIR/pain/energy feedback, receive a transparent rules-based adjustment, see progress, and follow editable energy/protein targets.

## Implemented outcomes

The bootstrap proves the engineering path with one narrow slice: a synthetic local identity can read and update its own minimal profile through a documented REST API and generated client used by an accessible Expo screen. Local infrastructure, lifecycle, logging, validation, tests, and production-readiness evidence must be real even though production capabilities remain incomplete.

Milestone 1 adds Auth0 OIDC integration paths for mobile and web, strict API token validation and scopes, and resumable owner-scoped onboarding. Onboarding captures adult attestation, versioned Terms/Privacy acceptance, timezone/units, goal, experience, schedule, equipment, optional diet preference, and a nonmedical safety boundary. It restores server progress, supports offline/error/retry states, uses idempotent writes, and re-gates when a required legal version changes.

## Users and acceptance

- Adults only; completion requires adult attestation and the current legal/safety acknowledgements.
- India-first defaults, but stored identifiers/timestamps and contracts remain globally usable.
- The profile screen covers loading, failure/retry, success, validation, conflict, and offline messaging.
- A clean checkout can run the documented local and verification commands without cloud credentials.
- Local identity bypasses fail before startup outside local/test.

## Excluded through Milestone 1

Workout generation/logging, progression, nutrition calculations, billing, notifications, analytics collection beyond no-op allowlisted events, operations UI, Terraform resources, production deployment, store releases, and any medical functionality. Auth0 dashboard/device verification and production identity operations remain release work, not automated claims.
