# Incident response runbook

Status: bootstrap template; production owners and contacts are not assigned.

1. Detect and open a restricted incident record; preserve timestamps, request IDs, alerts, and immutable evidence.
2. Triage affected data, identities, environments, vendors, time window, and ongoing risk without copying production data to lower environments.
3. Contain using least-disruptive credential revocation, access removal, traffic controls, or feature disablement; preserve evidence before destructive action.
4. Eradicate the cause, rotate affected credentials/keys, validate dependency and configuration integrity, and independently review authorization changes.
5. Recover from known-good artifacts/backups, validate data integrity and deletion state, monitor recurrence, and keep rollback available.
6. Engage qualified security, legal/privacy, executive, communications, insurer, vendor, and regulatory contacts according to the approved escalation matrix.
7. Document impact, decisions, notifications, lessons, corrective owners, and deadlines; rehearse the runbook before launch.

Credential/key compromise, data breach, ransomware, and third-party compromise require scenario-specific annexes and tested contacts before production.
