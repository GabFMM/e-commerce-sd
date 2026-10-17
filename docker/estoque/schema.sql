CREATE TABLE produtos (
    id serial PRIMARY KEY,
    nome VARCHAR(50) UNIQUE NOT NULL,
    categoria CHAR(1) NOT NULL CHECK (categoria IN ('A', 'B', 'C')),
    quantidade integer NOT NULL CHECK (quantidade >= 0),
    reservados integer DEFAULT 0 CHECK (reservados >= 0)
);
