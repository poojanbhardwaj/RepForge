# API conventions

- Routes are versioned under `/v1`; operational endpoints remain unversioned.
- JSON fields use lower camel case. Requests reject unsupported media types, unknown fields, trailing JSON, and oversized bodies.
- Errors use `{ "error": { "code", "message", "details", "traceId" } }`. Codes are stable; messages are safe for users.
- Growing collections use cursor pagination with explicit stable ordering.
- Retryable mutations define idempotency keys. Onboarding accepts exactly one `Idempotency-Key` containing 16–128 allowlisted ASCII characters; reuse with the same normalized request returns the original response and reuse with another request returns `409`.
- Authentication uses exactly one `Authorization: Bearer` value. Production paths use scoped OIDC access tokens; the fixed synthetic mapping is local/test only and is never a production protocol.
- Server-side authorization applies to every read/write regardless of client UI.
- Request cancellation propagates to storage. Server and dependency timeouts are bounded.
- Profile storage operations have a 10-second deadline and return one safe `504 profile_request_timeout` envelope on expiry; server socket deadlines remain independently enforced.
- Logs and traces use allowlisted metadata only and never contain authorization, bodies, sensitive values, or free text.
- CORS is denied by default. Local/test returns only the configured exact origin and does not enable credentials; nonlocal API configuration requires CORS to remain empty because web traffic uses the same-origin BFF.
- OpenAPI 3.1 is the contract source. Generated clients are committed and CI checks drift.
