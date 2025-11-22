DROP INDEX IF EXISTS idx_short_url;
CREATE INDEX idx_short_url ON shorts(short_url);
