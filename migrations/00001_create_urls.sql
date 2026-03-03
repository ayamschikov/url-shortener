-- +goose Up
CREATE TABLE urls (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(20) NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ
);

CREATE INDEX idx_urls_code ON urls (code);

-- +goose Down
DROP TABLE urls;
