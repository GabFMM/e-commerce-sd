CREATE TABLE IF NOT EXISTS interesses (
    email      TEXT PRIMARY KEY,
    categorias TEXT[] NOT NULL DEFAULT '{}'
    CHECK (categorias <@ ARRAY['A','B','C'])
);