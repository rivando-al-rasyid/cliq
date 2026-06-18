CREATE TABLE links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    origin_link text NOT NULL,
    slug varchar(50) NOT NULL UNIQUE,
    clicks integer NOT NULL DEFAULT 0,
    is_deleted boolean NOT NULL DEFAULT false,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NULL,
    deleted_at timestamp NULL,
    CONSTRAINT links_slug_length CHECK (char_length(slug) BETWEEN 3 AND 50),
    CONSTRAINT links_slug_format CHECK (slug ~ '^[A-Za-z0-9-]+$'),
    CONSTRAINT links_origin_link_format CHECK (origin_link ~* '^https?://'),
    CONSTRAINT links_clicks_non_negative CHECK (clicks >= 0),
    CONSTRAINT links_deleted_at_required_when_deleted CHECK (
        (is_deleted = false AND deleted_at IS NULL)
        OR
        (is_deleted = true AND deleted_at IS NOT NULL)
    )
);
