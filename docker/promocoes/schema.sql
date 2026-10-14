CREATE TABLE emails (
    id serial PRIMARY KEY,
    email text UNIQUE,
    categorias text[] CHECK (categorias <@ ARRAY['A', 'B', 'C']::text[])
);