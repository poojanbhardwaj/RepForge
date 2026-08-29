# RepForge bootstrap product requirements

## Product direction

RepForge helps adults in India follow safe strength training, log quickly, understand deterministic changes, and connect training with realistic nutrition. The product is for beginner-to-intermediate gym users and must support global expansion without becoming an all-sports application.

The repeated value loop is: receive an achievable workout, log it reliably including offline, give RIR/pain/energy feedback, receive a transparent rules-based adjustment, see progress, and follow editable energy/protein targets.

## Bootstrap outcome

The bootstrap proves the engineering path with one narrow slice: a synthetic local identity can read and update its own minimal profile through a documented REST API and generated client used by an accessible Expo screen. Local infrastructure, lifecycle, logging, validation, tests, and production-readiness evidence must be real even though production capabilities remain incomplete.

## Users and acceptance

- Adults only; the real adult gate is implemented in the onboarding milestone.
- India-first defaults, but stored identifiers/timestamps and contracts remain globally usable.
- The profile screen covers loading, failure/retry, success, validation, conflict, and offline messaging.
- A clean checkout can run the documented local and verification commands without cloud credentials.
- Local identity bypasses fail before startup outside local/test.

## Excluded from bootstrap

OIDC, onboarding, workout logging, progression, nutrition calculations, billing, notifications, analytics collection, operations UI, Terraform resources, production deployment, store builds, and any medical functionality.
