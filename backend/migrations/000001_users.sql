-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id uuid PRIMARY KEY,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'deletion_pending')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE identity_refs (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider text NOT NULL CHECK (char_length(provider) BETWEEN 1 AND 80),
    subject text NOT NULL CHECK (char_length(subject) BETWEEN 1 AND 255),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, subject)
);

CREATE INDEX identity_refs_user_id_idx ON identity_refs(user_id);

CREATE TABLE user_profiles (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name text NOT NULL CHECK (
        char_length(display_name) BETWEEN 1 AND 80
        AND display_name = btrim(display_name)
    ),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE consents (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document_key text NOT NULL CHECK (char_length(document_key) BETWEEN 1 AND 80),
    document_version text NOT NULL CHECK (char_length(document_version) BETWEEN 1 AND 80),
    source text NOT NULL CHECK (source IN ('mobile', 'web', 'support', 'synthetic_local')),
    accepted_at timestamptz NOT NULL,
    withdrawn_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (withdrawn_at IS NULL OR withdrawn_at >= accepted_at),
    UNIQUE (user_id, document_key, document_version)
);

CREATE INDEX consents_user_id_accepted_at_idx ON consents(user_id, accepted_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE consents;
DROP TABLE user_profiles;
DROP TABLE identity_refs;
DROP TABLE users;
-- +goose StatementEnd
