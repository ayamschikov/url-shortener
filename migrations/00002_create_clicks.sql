-- +goose Up
CREATE TABLE clicks (
    id         BIGSERIAL PRIMARY KEY,
    url_id     BIGINT NOT NULL REFERENCES urls(id),
    ip         VARCHAR(45),
    user_agent TEXT,
    referer    TEXT,
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clicks_url_id ON clicks (url_id);

-- +goose Down
DROP TABLE clicks;
