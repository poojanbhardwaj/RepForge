-- name: LockOnboardingIdentity :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(provider) || E'\x1f' || sqlc.arg(subject), 0));

-- name: GetIdentityUser :one
SELECT i.user_id, u.status
FROM identity_refs i
JOIN users u ON u.id = i.user_id
WHERE i.provider = sqlc.arg(provider)
  AND i.subject = sqlc.arg(subject);

-- name: CreateUser :exec
INSERT INTO users (id) VALUES (sqlc.arg(id));

-- name: CreateIdentity :exec
INSERT INTO identity_refs (id, user_id, provider, subject)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(provider), sqlc.arg(subject));

-- name: CreateDefaultProfile :exec
INSERT INTO user_profiles (user_id, display_name)
VALUES (sqlc.arg(user_id), 'Athlete')
ON CONFLICT (user_id) DO NOTHING;

-- name: CreateOnboarding :exec
INSERT INTO user_onboarding (user_id) VALUES (sqlc.arg(user_id))
ON CONFLICT (user_id) DO NOTHING;

-- name: LockOnboarding :one
SELECT * FROM user_onboarding
WHERE user_id = sqlc.arg(user_id)
FOR UPDATE;

-- name: GetOnboardingByIdentity :one
SELECT o.*
FROM identity_refs i
JOIN users u ON u.id = i.user_id
JOIN user_onboarding o ON o.user_id = u.id
WHERE i.provider = sqlc.arg(provider)
  AND i.subject = sqlc.arg(subject)
  AND u.status = 'active';

-- name: UpdateOnboarding :one
UPDATE user_onboarding
SET adult_attested_at = sqlc.arg(adult_attested_at),
    terms_version = sqlc.arg(terms_version),
    terms_accepted_at = sqlc.arg(terms_accepted_at),
    privacy_version = sqlc.arg(privacy_version),
    privacy_accepted_at = sqlc.arg(privacy_accepted_at),
    timezone = sqlc.arg(timezone),
    units = sqlc.arg(units),
    primary_goal = sqlc.arg(primary_goal),
    experience_level = sqlc.arg(experience_level),
    weekly_availability = sqlc.arg(weekly_availability),
    session_duration_minutes = sqlc.arg(session_duration_minutes),
    equipment_access = sqlc.arg(equipment_access),
    diet_preference = sqlc.arg(diet_preference),
    safety_acknowledged_at = sqlc.arg(safety_acknowledged_at),
    current_step = sqlc.arg(current_step),
    state = sqlc.arg(state),
    completed_at = sqlc.arg(completed_at),
    version = version + 1,
    updated_at = sqlc.arg(updated_at)
WHERE user_id = sqlc.arg(user_id)
RETURNING *;

-- name: CreateConsent :exec
INSERT INTO consents (id, user_id, document_key, document_version, source, accepted_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(document_key), sqlc.arg(document_version), sqlc.arg(source), sqlc.arg(accepted_at))
ON CONFLICT (user_id, document_key, document_version) DO NOTHING;

-- name: GetIdempotencyRecord :one
SELECT request_hash, response_snapshot
FROM onboarding_idempotency
WHERE user_id = sqlc.arg(user_id) AND idempotency_key = sqlc.arg(idempotency_key);

-- name: CreateIdempotencyRecord :exec
INSERT INTO onboarding_idempotency (user_id, idempotency_key, request_hash, response_snapshot)
VALUES (sqlc.arg(user_id), sqlc.arg(idempotency_key), sqlc.arg(request_hash), sqlc.arg(response_snapshot));
