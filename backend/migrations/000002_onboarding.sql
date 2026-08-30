-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_onboarding (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    adult_attested_at timestamptz,
    terms_version text CHECK (terms_version IS NULL OR char_length(terms_version) BETWEEN 1 AND 80),
    terms_accepted_at timestamptz,
    privacy_version text CHECK (privacy_version IS NULL OR char_length(privacy_version) BETWEEN 1 AND 80),
    privacy_accepted_at timestamptz,
    timezone text CHECK (timezone IS NULL OR char_length(timezone) BETWEEN 1 AND 64),
    units text CHECK (units IS NULL OR units IN ('metric', 'imperial')),
    primary_goal text CHECK (primary_goal IS NULL OR primary_goal IN ('strength', 'muscle', 'general_fitness')),
    experience_level text CHECK (experience_level IS NULL OR experience_level IN ('beginner', 'intermediate')),
    weekly_availability smallint CHECK (weekly_availability IS NULL OR weekly_availability BETWEEN 1 AND 7),
    session_duration_minutes smallint CHECK (session_duration_minutes IS NULL OR session_duration_minutes BETWEEN 15 AND 180),
    equipment_access text[] CHECK (equipment_access IS NULL OR cardinality(equipment_access) BETWEEN 1 AND 8),
    diet_preference text CHECK (diet_preference IS NULL OR diet_preference IN ('vegetarian', 'eggetarian', 'vegan', 'omnivore')),
    safety_acknowledged_at timestamptz,
    current_step smallint NOT NULL DEFAULT 1 CHECK (current_step BETWEEN 1 AND 6),
    state text NOT NULL DEFAULT 'in_progress' CHECK (state IN ('in_progress', 'complete')),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((terms_version IS NULL) = (terms_accepted_at IS NULL)),
    CHECK ((privacy_version IS NULL) = (privacy_accepted_at IS NULL)),
    CHECK ((state = 'complete') = (completed_at IS NOT NULL)),
    CHECK (
        state <> 'complete'
        OR (
            adult_attested_at IS NOT NULL
            AND terms_accepted_at IS NOT NULL
            AND privacy_accepted_at IS NOT NULL
            AND timezone IS NOT NULL
            AND units IS NOT NULL
            AND primary_goal IS NOT NULL
            AND experience_level IS NOT NULL
            AND weekly_availability IS NOT NULL
            AND session_duration_minutes IS NOT NULL
            AND equipment_access IS NOT NULL
            AND safety_acknowledged_at IS NOT NULL
        )
    )
);

CREATE TABLE onboarding_idempotency (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idempotency_key text NOT NULL CHECK (char_length(idempotency_key) BETWEEN 16 AND 128),
    request_hash bytea NOT NULL CHECK (octet_length(request_hash) = 32),
    response_snapshot jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, idempotency_key)
);

CREATE INDEX onboarding_idempotency_created_at_idx ON onboarding_idempotency(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE onboarding_idempotency;
DROP TABLE user_onboarding;
-- +goose StatementEnd
