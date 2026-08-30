# Auth0 local verification runbook

This is a manual development procedure. Repository automation does not create, inspect, or modify an Auth0 tenant. Never commit or paste client secrets, cookie secrets, tokens, authorization codes, or `.env.local` contents into logs, issues, screenshots, or chat.

## Applications and API

Use a dedicated nonproduction tenant and separate applications:

- Native application for the Expo development build, using Authorization Code + PKCE and refresh-token rotation.
- Regular Web Application for Next.js; its client secret is server-only.
- API with the configured HTTPS audience, RFC 9068 access-token profile, and `profile:read` / `profile:write` permissions.

The native package identifiers are `local.repforge.bootstrap` on Android and iOS, and the custom scheme is `repforge`. The native plugin domain and public domain variable must match the tenant. Register the SDK-generated callback/logout forms for the configured domain:

- `repforge://<tenant-domain>/android/local.repforge.bootstrap/callback`
- `repforge://<tenant-domain>/ios/local.repforge.bootstrap/callback`

For the local web application, register:

- callback: `http://127.0.0.1:3000/auth/callback`
- logout return and web origin: `http://127.0.0.1:3000`

Do not add wildcard callbacks/origins or `localhost` aliases. Require verified email according to the tenant policy. Confirm the API issues RS256 RFC 9068 access tokens with `typ: at+jwt`, `client_id`, the configured audience, authorized application ID, and requested scopes.

## Local configuration

If `.env.local` predates Milestone 1, add the server-only Auth0 variables shown by name in `.env.example`; do not print the file. Supply the native and web client IDs and the web client secret manually. Generate a fresh 32-byte web session secret locally. Keep all secrets only in the ignored local file or an approved secret manager.

For the API, set `AUTH_MODE=oidc` and ensure issuer, audience, and both client IDs exactly match. For mobile, set `EXPO_PUBLIC_AUTH_MODE=oidc`, remove `EXPO_PUBLIC_DEV_AUTH_TOKEN`, and set only the public domain/client ID/audience. A remote or device-reachable API must use a canonical HTTPS origin; the literal-loopback HTTP exception is emulator/simulator development only.

Rebuild the development client after plugin, package identifier, scheme, or domain changes. Expo Go is not sufficient for the native Auth0 module.

## Manual cases

Record sanitized pass/fail evidence without tokens or user-entered values:

1. New verified user signs in on Android and iOS, returns through PKCE, and resumes after process death.
2. Cancel, access-denied/unverified-email, offline login, refresh while offline, expired refresh token, and logout/browser-logout failure show the defined recoverable states.
3. Logout removes local credentials and cached user data; a second user cannot see the first user’s query cache.
4. Web login returns only to `/app`; protected pages and BFF return safe failures for missing/corrupt/expired sessions.
5. Browser requests with a wrong/missing Origin, cross-site Fetch Metadata, invalid JSON/content type, oversized body, or invalid idempotency key are rejected before API mutation.
6. Mobile and web can restore partial onboarding, retry the same idempotency key, complete onboarding, clear the optional diet value, and become incomplete again when Terms or Privacy version changes.
7. Tokens with wrong issuer, audience, client, algorithm/key, expiry/issued-at/not-before, critical header, malformed scope, or missing required scope fail with no sensitive logs.

This milestone is not production identity approval. Production callback domains, custom domains, MFA, attack protection, refresh-token policy, tenant administrators, log export/retention, breach response, and data-processing terms require separate security/privacy ownership and evidence.
