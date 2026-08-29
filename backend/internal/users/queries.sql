-- name: GetProfileByIdentity :one
SELECT u.id AS user_id,
       p.display_name,
       p.version,
       p.created_at,
       p.updated_at
FROM identity_refs i
JOIN users u ON u.id = i.user_id
JOIN user_profiles p ON p.user_id = u.id
WHERE i.provider = sqlc.arg(provider)
  AND i.subject = sqlc.arg(subject)
  AND u.status = 'active';

-- name: ListConsentsByUser :many
SELECT id, document_key, document_version, source, accepted_at, withdrawn_at
FROM consents
WHERE user_id = sqlc.arg(user_id)
ORDER BY accepted_at DESC, id DESC;

-- name: UpdateProfileByIdentity :one
UPDATE user_profiles AS p
SET display_name = sqlc.arg(display_name),
    version = p.version + 1,
    updated_at = now()
FROM identity_refs AS i, users AS u
WHERE i.provider = sqlc.arg(provider)
  AND i.subject = sqlc.arg(subject)
  AND i.user_id = u.id
  AND u.status = 'active'
  AND p.user_id = u.id
  AND p.version = sqlc.arg(expected_version)
RETURNING p.user_id, p.display_name, p.version, p.created_at, p.updated_at;

-- name: GetUserIDByIdentity :one
SELECT user_id
FROM identity_refs
WHERE provider = sqlc.arg(provider) AND subject = sqlc.arg(subject);

-- name: CreateUser :exec
INSERT INTO users (id) VALUES (sqlc.arg(id));

-- name: CreateIdentityRef :exec
INSERT INTO identity_refs (id, user_id, provider, subject)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(provider), sqlc.arg(subject));

-- name: CreateProfile :exec
INSERT INTO user_profiles (user_id, display_name)
VALUES (sqlc.arg(user_id), sqlc.arg(display_name))
ON CONFLICT (user_id) DO NOTHING;

-- name: CreateConsent :exec
INSERT INTO consents (id, user_id, document_key, document_version, source, accepted_at)
VALUES (
    sqlc.arg(id),
    sqlc.arg(user_id),
    sqlc.arg(document_key),
    sqlc.arg(document_version),
    sqlc.arg(source),
    sqlc.arg(accepted_at)
)
ON CONFLICT (user_id, document_key, document_version) DO NOTHING;
