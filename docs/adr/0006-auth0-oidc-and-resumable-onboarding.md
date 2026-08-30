# ADR 0006: Auth0 OIDC paths and resumable onboarding

Status: accepted for Milestone 1 local implementation; live tenant/device verification pending.

## Context

The bootstrap’s deterministic bearer mapping cannot represent a production identity flow. Mobile needs native Authorization Code + PKCE and process-death recovery. Web needs server-managed sessions without exposing API tokens to browser JavaScript. Onboarding must survive interruption, retain owner isolation, and safely retry mobile/network mutations while legal document versions can change.

## Decision

- Use Auth0 as the selected OIDC integration target. Mobile uses the native Auth0 SDK with Authorization Code + PKCE and refresh credentials in platform-protected storage. Web uses `@auth0/nextjs-auth0` v4 with encrypted HttpOnly stateless cookies and a same-origin BFF. No dashboard automation or production configuration is included.
- Keep deterministic authentication only for `local` and `test`; configuration fails closed elsewhere. The configured API uses Auth0's RFC 9068 profile, so the Go API requires `typ: at+jwt` and `client_id`, then validates canonical same-origin discovery/JWKS, RS256 keys, issuer, audience, authorized mobile/web client, signature, temporal claims, subject, optional verified-email claim, and scopes. Auth0's default `typ: JWT` / `azp` profile is rejected.
- Do not place API access tokens in browser code or logs. The BFF applies exact-origin/Fetch Metadata checks, strict media/idempotency syntax, bounded request/upstream bodies, fixed upstream routing, and no-store responses.
- Store onboarding separately from profile, keyed by the application user. First login provisions user, identity, default profile, and onboarding state in one transaction serialized by provider/subject. All reads/writes resolve ownership from the authenticated identity.
- Require a syntactically strict, owner-scoped idempotency key for onboarding updates. Hash the normalized request plus client source and required legal versions, lock the onboarding row, and persist the authoritative response snapshot. Same-key/same-request replays return that snapshot; mismatches return conflict.
- Preserve first adult/safety acceptance timestamps. Record every newly accepted Terms/Privacy document version in consent evidence. Presentation derives completion against current configured versions, so a version change re-gates without deleting prior evidence.
- Keep optional diet preference nullable and explicitly clearable. Do not collect diagnoses, injuries, medications, body measurements, or free-text health data in this milestone.

## Consequences

The mobile app requires a native development build rather than Expo Go. Live Auth0 flows require manual callback/logout/API configuration and device testing. Web depends on server-only secrets and cannot provide protected routes when configuration is missing. JWKS availability and rotation are external dependencies, mitigated by bounded caching, same-key refresh, timeouts, response limits, and fail-closed behavior. Rate limiting, production tenant controls, staff access/MFA, centralized Auth0 logs, device attestation, and independent mobile testing remain release gates.

Workout generation and logging are intentionally outside this decision.
