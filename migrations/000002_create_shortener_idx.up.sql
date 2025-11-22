DROP INDEX IF EXISTS idx_short_url;
CREATE UNIQUE INDEX IF NOT EXISTS idx_short_url ON shorts (short_url);