CREATE TABLE produtos (
    id serial PRIMARY KEY,
    nome VARCHAR(50) UNIQUE NOT NULL,
    categoria CHAR(1) NOT NULL CHECK (categoria IN ('A', 'B', 'C')),
    quantidade integer NOT NULL CHECK (quantidade >= 0),
    reservados integer DEFAULT 0 CHECK (reservados >= 0)
);

CREATE TABLE pedidos (
    id serial PRIMARY KEY,
    situacao VARCHAR(20) NOT NULL CHECK (situacao IN ('CRIADO', 'ESTOQUE_DISPONIVEL', 'ESTOQUE_INDISPONIVEL', 'PAGAMENTO_APROVADO', 'PAGAMENTO_RECUSADO' || 'ENVIADO'))
);

CREATE TABLE pedidos_produtos (
    pedido_id integer NOT NULL,
    produto_id integer NOT NULL,
    quantidade integer NOT NULL CHECK (quantidade > 0),
    PRIMARY KEY (pedido_id, produto_id),
    FOREIGN KEY (pedido_id) REFERENCES pedidos(id) ON DELETE CASCADE,
    FOREIGN KEY (produto_id) REFERENCES produtos(id)
);