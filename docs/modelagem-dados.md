# Modelagem dos dados

**Banco de dados:** PostgreSQL

### Visualização

   +-------------------+                     +-------------------+ 
   |     produtos      |                     |      pedidos      | 
   +-------------------+                     +-------------------+ 
   | PK  id            |                     | PK  id            | 
   |     categoria     |                     |     situacao      | 
   |     quantidade    |                     +-------------------+ 
   |     reservados    |                               | 1
   +-------------------+                               |
             | 1                                       |
             |                                         |
             |               (1:N)                     | (1:N)
             +-------------------++--------------------+
                                 ||
                                 \/
                       +--------------------+
                       |  pedidos_produtos  |
                       +--------------------+
                       | PK,FK1  pedido_id  |
                       | PK,FK2  produto_id |
                       |         quantidade |
                       +--------------------+


### Tabela: `produtos`

Armazena os produtos disponíveis no estoque.

**Colunas:**

* `id` — `serial`, **PRIMARY KEY**
* `categoria` — `char(1)`, **NOT NULL**, com valores permitidos `A`, `B` ou `C`
* `quantidade` — `integer`, **NOT NULL**, com valor mínimo igual a `0`
* `reservados` — `integer`, **DEFAULT 0**, com valor mínimo igual a `0`

### Tabela: `pedidos`

Armazena os pedidos criados e atualizados no ms-principal.

**Colunas:**

* `id` — `serial`, **PRIMARY KEY**
* `situacao` - `varchar()`, **NOT NULL**, com valores permitidos `CRIADO`, `ESTOQUE_DISPONIVEL`, `ESTOQUE_INDISPONIVEL`, `PAGAMENTO_APROVADO`, `PAGAMENTO_RECUSADO` ou `ENVIADO`

**Explicação das situacoes:**
- `CRIADO`: o pedido foi criado e foi encaminhado para o ms-estoque;
- `ESTOQUE_DISPONIVEL`: o estoque possui a quantidade requisitida de cada produto no pedido; e foi encaminhado para o ms-pagamento;
- `ESTOQUE_INDISPONIVEL`: o estoque não possui a quantidade requisitida de um dos produtos no pedido; e foi encaminhado para o ms-principal;
- `PAGAMENTO_APROVADO`: o pagamento do pedido foi aprovado e foi encaminhado para o ms-entrega;
- `PAGAMENTO_RECUSADO`: o pagamento do pedido foi recusado e foi encaminhado para o ms-principal;
- `ENVIADO`: o pedido foi enviado para o usuário e foi encaminhado para o ms-principal.

### Tabela: `pedidos_produtos`

Relaciona os pedidos aos produtos e registra a quantidade de cada produto reservada para o pedido.

**Colunas:**

* `pedido_id` — `integer`, **FOREIGN KEY** para `pedidos(id)`
* `produto_id` — `integer`, **FOREIGN KEY** para `produtos(id)` com **ON DELETE CASCADE**
* `quantidade` — `integer`, **NOT NULL**, com valor mínimo igual a `1`

A chave primária da tabela será composta por `pedido_id` e `produto_id`.

### Relacionamentos

* Um **pedido** pode possuir vários **produtos**.
* Um **produto** pode estar presente em vários **pedidos**.
* A tabela `pedidos_produtos` representa o relacionamento **N:M** entre `pedidos` e `produtos`.