# Dependency record

Verified 2026-08-29 from official release documentation, package registries, the locked graphs, and the locally pulled container metadata. Exact transitive versions are recorded in `go.sum` and `pnpm-lock.yaml`.

| Component            |         Pin/baseline | Reason                                                          |
| -------------------- | -------------------: | --------------------------------------------------------------- |
| Go                   |     1.26.7 toolchain | Supported patched line; includes the post-1.26.6 `net/http` fix |
| Node.js              | 24.20.0 LTS baseline | Supported LTS; local 24.16.0 remains compatible for bootstrap   |
| pnpm                 |              11.24.0 | Current supported v11 release                                   |
| Expo / Router        |    57.0.17 / 57.0.17 | Current SDK 57 patches                                          |
| React / React Native |      19.2.8 / 0.86.3 | Framework-compatible releases                                   |
| Next.js              |               16.3.3 | Active LTS with August 2026 security fixes                      |
| Auth0 Next.js SDK    |               4.28.0 | Server session, callback middleware, and server token refresh   |
| React Native Auth0   |               5.11.0 | Native Authorization Code + PKCE and protected credentials      |
| Go JWT               |                5.3.1 | Strict access-token parsing and RS256 verification              |
| TypeScript           |                5.9.3 | Expo 57 template-compatible toolchain                           |
| PostgreSQL           |                 18.6 | Current supported patch                                         |
| Redis                |                8.2.9 | Patched extended-support line; local cache only                 |
| Garage               |                2.3.0 | Maintained local S3-compatible service; never production        |
| Trivy                |               0.73.0 | Digest-pinned local/CI container OS vulnerability scanner       |
| Gitleaks             |               8.28.0 | Digest-pinned working-tree and Git-history secret scanner       |
| sqlc                 |               1.31.1 | Typed SQL generation                                            |
| Transitive `uuid`    |               11.1.1 | Patched override for Expo's build-time `xcode` dependency       |

Go libraries are minimized to PostgreSQL, migrations, UUIDv7, Redis/S3 adapters, and OpenTelemetry APIs. TypeScript dependencies are exact top-level pins and lifecycle scripts are denied unless allowlisted.

MinIO is intentionally excluded because its community repository is archived. Garage uses a single-node local mode; AWS S3 is the production reference.

Locally verified image digests on 2026-08-29:

- `postgres:18.6-alpine@sha256:d3e1620b530c944afa6e887d22eb899824da68e19c52024bf98f5220c88a65b2`
- `redis:8.2.9-alpine@sha256:30abb90e62f14b737010746def3ba99cc79fe19dcdb3d37b41f21fc62e7da19d`
- `dxflrs/garage:v2.3.0@sha256:866bd13ed2038ba7e7190e840482bc27234c4afaf77be8cfa439ae088c1e4690`
- `aquasec/trivy:0.73.0@sha256:7cced7cae583819fc7806d4cbc0dbbc7cad18b99f7d3e235192e6da8c091045c`
- `zricethezav/gitleaks:v8.28.0@sha256:cdbb7c955abce02001a9f6c9f602fb195b7fadc1e812065883f695d1eeaba854`

Compose uses readable version tags plus mandatory immutable manifest digests. `pnpm run images:verify` resolves every digest from its public registry before running the digest-pinned Trivy scanner against OS packages. Each temporary local CVE waiver is mounted only for its exact PostgreSQL or Redis image/digest; changed or unrelated images receive no waiver. Production scans never mount either local waiver. PostgreSQL and Redis publish OCI source/revision metadata; Garage's older manifest list does not publish equivalent provenance attestations.

The 2026-08-29 local scan inventoried 53 PostgreSQL and 22 Redis OS packages and reported zero unsuppressed high/critical OS findings. `CVE-2026-14456` in OpenSSL 3.5.7-r0 is temporarily excepted through 2026-09-30 only for those exact local manifests because it is specific to QUIC server operation and these local services expose TCP on loopback only; an upstream rebuild with OpenSSL 3.5.8-r0 must replace the exceptions. The waiver-free production scan reports two unsuppressed findings for that CVE in each image. Trivy returns no OS/package inventory for Garage. The local gate reports that as missing coverage and permits Garage only as a local fixture; `pnpm run images:production-gate` fails on those unwaived findings, Garage's missing coverage/local-only status, and absent release SBOM/publisher provenance evidence.

`pnpm run secrets` verifies the Gitleaks registry digest and scanner canary, then runs two disposable regressions: an unrelated token beside an exact generated `.env.local` value must be detected, and a token removed from the current tree but retained in a temporary committed repository's prior commit must be detected by the same `gitleaks git` command used by CI. Gitleaks has no file-wide or path-wide credential exclusion; `.gitleaks.toml` excludes only generated dependency/build outputs from the working-tree input. The redacted working-tree report accepts only generic-key findings on validated exact generated fixture lines; every other finding fails. The deterministic Garage RPC fixture retains one line-scoped inline annotation. The project now has a reachable bootstrap commit, so security scans both project history and the current working tree.

License and vulnerability status must be rechecked during every dependency update. Docker Desktop licensing is the contributor/organization's responsibility.
