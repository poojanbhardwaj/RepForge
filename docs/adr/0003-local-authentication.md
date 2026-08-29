# ADR 0003: Fail-closed local authentication

Status: accepted for bootstrap; superseded on real application paths by the authentication milestone.

A development authenticator maps one configured bearer token to one fixed synthetic subject. Configuration accepts it only when `APP_ENV` is `local` or `test`; all other combinations fail before listening. Arbitrary subject headers, passwords, unsigned JWTs, and production fallback are rejected.
