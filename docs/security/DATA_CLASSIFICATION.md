# Data classification

| Class                             | Examples                                                 | Handling                                                                            |
| --------------------------------- | -------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| Public                            | Marketing copy, exercise content with verified rights    | Integrity controls; safe for public delivery                                        |
| Internal                          | Build metadata, non-sensitive operational configuration  | Authenticated staff/repository access                                               |
| Confidential                      | Email/identity references, display name, support records | Least privilege, encryption, minimization, audited access                           |
| Sensitive fitness/health-adjacent | Workout, pain, body, readiness, nutrition behavior       | Confidential controls plus strict purpose/consent, short retention, export/deletion |
| Restricted secret                 | Tokens, keys, payment/webhook secrets, KMS material      | Managed secret storage, rotation, never logged or placed in clients/source          |

Milestone 1 stores only synthetic identity/profile/onboarding/consent/idempotency data locally. OIDC code can process real login identifiers only during explicitly authorized manual testing. Production or personal fitness data is prohibited in local development, CI, screenshots, logs, and AI prompts.
