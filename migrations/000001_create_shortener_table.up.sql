CREATE TABLE shorts (
    uuid uuid PRIMARY KEY,
    short_url text NOT NULL,
    original_url text NOT NULL
);

CREATE INDEX idx_short_url ON shorts(short_url);
