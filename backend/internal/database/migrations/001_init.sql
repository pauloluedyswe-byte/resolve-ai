CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT        NOT NULL,
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL CHECK (role IN ('solicitante', 'gestor')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE categories (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

INSERT INTO categories (name) VALUES
    ('Iluminação'),
    ('Equipamentos quebrados'),
    ('Acessibilidade'),
    ('Limpeza'),
    ('Vazamentos'),
    ('Segurança'),
    ('Manutenção'),
    ('Outros');

CREATE TABLE occurrences (
    id             BIGSERIAL PRIMARY KEY,
    title          TEXT        NOT NULL,
    description    TEXT        NOT NULL,
    category_id    BIGINT      NOT NULL REFERENCES categories(id),
    location       TEXT        NOT NULL,
    image_url      TEXT,
    priority       TEXT        NOT NULL DEFAULT 'media'
                   CHECK (priority IN ('baixa', 'media', 'alta', 'critica')),
    status         TEXT        NOT NULL DEFAULT 'aberta'
                   CHECK (status IN ('aberta', 'em_analise', 'em_atendimento', 'resolvida', 'cancelada')),
    requester_id   BIGINT      NOT NULL REFERENCES users(id),
    assignee_id    BIGINT      REFERENCES users(id),
    solution       TEXT,
    rating         SMALLINT    CHECK (rating BETWEEN 1 AND 5),
    rating_comment TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at    TIMESTAMPTZ
);

CREATE INDEX idx_occurrences_requester ON occurrences(requester_id);
CREATE INDEX idx_occurrences_status    ON occurrences(status);
CREATE INDEX idx_occurrences_category  ON occurrences(category_id);
CREATE INDEX idx_occurrences_priority  ON occurrences(priority);

CREATE TABLE comments (
    id            BIGSERIAL PRIMARY KEY,
    occurrence_id BIGINT      NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    author_id     BIGINT      NOT NULL REFERENCES users(id),
    body          TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comments_occurrence ON comments(occurrence_id);

-- Histórico auditável: toda mudança de status gera uma linha aqui.
CREATE TABLE status_history (
    id            BIGSERIAL PRIMARY KEY,
    occurrence_id BIGINT      NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    from_status   TEXT,
    to_status     TEXT        NOT NULL,
    changed_by    BIGINT      NOT NULL REFERENCES users(id),
    note          TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_status_history_occurrence ON status_history(occurrence_id);
