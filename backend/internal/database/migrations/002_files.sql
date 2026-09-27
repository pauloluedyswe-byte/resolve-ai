-- Imagens anexadas ficam no banco: funciona em hosts com disco efêmero (ex.: Render free).
CREATE TABLE files (
    name         TEXT        PRIMARY KEY,
    content_type TEXT        NOT NULL,
    data         BYTEA       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
