CREATE INDEX idx_users_email ON users(email);

CREATE INDEX idx_profiles_user_id ON profiles(user_id);

CREATE INDEX idx_tokens_token ON tokens(token);
CREATE INDEX idx_tokens_user_id ON tokens(user_id);
CREATE INDEX idx_tokens_active_lookup ON tokens(token, type, expires_at)
    WHERE is_revoked = false;

CREATE INDEX idx_links_user_active_created_at ON links(user_id, is_deleted, created_at DESC);
CREATE INDEX idx_links_slug_active ON links(slug)
    WHERE is_deleted = false;
CREATE INDEX idx_links_created_at ON links(created_at DESC);
