# API conventions

- Routes are versioned under `/v1`; operational endpoints remain unversioned.
- JSON fields use lower camel case. Requests reject unsupported media types, unknown fields, trailing JSON, and oversized bodies.
- Errors use `{ "error": { "code", "message", "details", "traceId" } }`. Codes are stable; messages are safe for users.
- Growing collections use cursor pagination with explicit stable ordering.
- Retryable creates require idempotency keys. Offline-editable resources include optimistic versions.
- Authentication uses `Authorization: Bearer`. The local token is a fixed synthetic mapping and is never a production protocol.
- Server-side authorization applies to every read/write regardless of client UI.
- Request cancellation propagates to storage. Server and dependency timeouts are bounded.
- Profile storage operations have a 10-second deadline and return one safe `504 profile_request_timeout` envelope on expiry; server socket deadlines remain independently enforced.
- Logs and traces use allowlisted metadata only and never contain authorization, bodies, sensitive values, or free text.
- CORS is denied by default and allowed origins must be explicit per environment.
- OpenAPI 3.1 is the contract source. Generated clients are committed and CI checks drift.
