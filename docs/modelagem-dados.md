# Modelagem dos dados

**Banco de dados:** PostgreSQL

### Tabela: `produtos`

Armazena os produtos disponíveis no estoque.

**Colunas:**

* `id` — `serial`, **PRIMARY KEY**
* `categoria` — `char(1)`, **NOT NULL**, com valores permitidos `A`, `B` ou `C`
* `quantidade` — `integer`, **NOT NULL**, com valor mínimo igual a `0`

### Tabela: `pedidos`

Armazena os pedidos recebidos pelo microsserviço Estoque, permitindo manter o estado necessário para controlar as reservas realizadas.

**Colunas:**

* `id` — `serial`, **PRIMARY KEY**

### Tabela: `pedidos_produtos`

Relaciona os pedidos aos produtos e registra a quantidade de cada produto reservada para o pedido.

**Colunas:**

* `pedido_id` — `integer`, **FOREIGN KEY** para `pedidos(id)`
* `produto_id` — `integer`, **FOREIGN KEY** para `produtos(id)`
* `quantidade` — `integer`, **NOT NULL**, com valor mínimo igual a `1`

A chave primária da tabela será composta por `pedido_id` e `produto_id`.

### Relacionamentos

* Um **pedido** pode possuir vários **itens**.
* Um **produto** pode estar presente em vários **pedidos**.
* A tabela `itens_pedido` representa o relacionamento **N:N** entre `pedidos` e `produtos`.

Essa estrutura permite que o microsserviço Estoque registre quais produtos foram reservados para cada pedido. Dessa forma, caso seja recebido posteriormente um evento `pedido.excluido`, o serviço poderá identificar as quantidades anteriormente reservadas e devolvê-las ao estoque.
