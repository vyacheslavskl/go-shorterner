CREATE TABLE user_urls (
    user_id uuid NOT NULL,
    url_uuid uuid NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_user_urls_user_id ON user_urls(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_url ON user_urls (user_id, url_uuid);
