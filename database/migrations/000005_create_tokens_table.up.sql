CREATE TABLE tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token text NOT NULL UNIQUE,
    type token_type NOT NULL,
    expires_at timestamp NOT NULL,
    is_revoked boolean NOT NULL DEFAULT false,
    created_at timestamp NOT NULL DEFAULT now(),
    CONSTRAINT tokens_expires_after_created CHECK (expires_at > created_at)
);
