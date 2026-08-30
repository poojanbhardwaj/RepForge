# Data flows and vendor inventory

## Local and optional OIDC flows

```text
Synthetic user -> Expo app -> localhost Go API -> local PostgreSQL
                                      |-> local Redis adapter test
                                      |-> local Garage S3 adapter test

OIDC mobile user -> Auth0 hosted authorization -> native protected credentials
  -> access token -> Go API -> PostgreSQL

OIDC web user -> Auth0 hosted authorization -> encrypted HttpOnly Next.js session
  -> server-only access token -> Go API -> PostgreSQL
```

All default local services bind to loopback. No telemetry exporter is enabled. The generated client receives only the synthetic local token in explicit development mode. The Auth0 paths are implemented but require manual tenant configuration; no dashboard or production connection was inspected or changed during this milestone.

## Inventory

| System/vendor                                     | Current role                | Data                                              | Production status/removal                                                                                                             |
| ------------------------------------------------- | --------------------------- | ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| PostgreSQL image                                  | Local primary store         | Synthetic identity/profile/onboarding/consent     | Replace with managed RDS after review                                                                                                 |
| Redis image                                       | Adapter/integration test    | Synthetic test keys                               | Replace with managed Redis when required                                                                                              |
| Garage                                            | Local S3 compatibility      | Synthetic test objects                            | Never production; remove container/volume                                                                                             |
| Auth0                                             | Selected OIDC integration   | Login identifiers, auth/session security metadata | Nonproduction manual verification pending; DPA, region, retention, tenant controls, MFA/admin access and production approval required |
| GitHub Actions                                    | CI reference                | Source and synthetic fixtures                     | Repository connection/configuration pending                                                                                           |
| AWS                                               | Architecture reference only | None                                              | No account, keys, or resources configured                                                                                             |
| Expo/Apple/Google/billing/analytics/error vendors | Not connected               | None                                              | Separate vendor, DPA, location, retention, and permission review required                                                             |

No production health/fitness data may be sent to an LLM without a separately approved, minimized, consented, legally and contractually reviewed design.
