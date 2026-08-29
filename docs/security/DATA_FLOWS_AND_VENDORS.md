# Data flows and vendor inventory

## Bootstrap flow

```text
Synthetic user -> Expo app -> localhost Go API -> local PostgreSQL
                                      |-> local Redis adapter test
                                      |-> local Garage S3 adapter test
```

All local services bind to loopback. No telemetry exporter or third-party API is enabled. The generated client receives only the synthetic local token during development.

## Inventory

| System/vendor                                          | Bootstrap role              | Data                               | Production status/removal                                                 |
| ------------------------------------------------------ | --------------------------- | ---------------------------------- | ------------------------------------------------------------------------- |
| PostgreSQL image                                       | Local primary store         | Synthetic identity/profile/consent | Replace with managed RDS after review                                     |
| Redis image                                            | Adapter/integration test    | Synthetic test keys                | Replace with managed Redis when required                                  |
| Garage                                                 | Local S3 compatibility      | Synthetic test objects             | Never production; remove container/volume                                 |
| GitHub Actions                                         | CI reference                | Source and synthetic fixtures      | Repository connection/configuration pending                               |
| AWS                                                    | Architecture reference only | None                               | No account, keys, or resources configured                                 |
| Expo/Apple/Google/OIDC/billing/analytics/error vendors | Not connected               | None                               | Separate vendor, DPA, location, retention, and permission review required |

No production health/fitness data may be sent to an LLM without a separately approved, minimized, consented, legally and contractually reviewed design.
