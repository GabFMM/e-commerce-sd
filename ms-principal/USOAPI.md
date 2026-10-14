# Uso da API

API REST do **MS Principal** (único ponto de entrada do frontend).

- **Base URL:** `http://localhost:8080/api`
- **Content-Type:** `application/json`
- **Erros:** `{"erro": "mensagem"}`

## Pré-requisitos

- RabbitMQ rodando (`localhost:5672`) e filas configuradas.
- MS Principal, Estoque, Pagamento, Entrega e Promoções no ar.


## Chave do Resend (e-mails de promoção)

O MS Promoções lê a chave de um `.env` na inicialização:

```env
RESEND_API_KEY=re_xxxxxxxxxxxx
```

> **Importante:** sem um domínio verificado, o Resend só envia e-mails para o **mesmo e-mail com que você criou a conta**. Cadastre esse e-mail nas promoções para testar. Ao trocar a chave, reinicie o MS Promoções.

---

## Endpoints

| Método | Rota                       | Descrição                    | Sucesso |
|--------|----------------------------|------------------------------|---------|
| GET    | `/produtos`                | Lista produtos               | 200     |
| POST   | `/pedidos`                 | Cria pedido                  | 201     |
| GET    | `/pedidos`                 | Lista pedidos (`?situacao=`) | 200     |
| GET    | `/pedidos/{id}`            | Consulta pedido              | 200     |
| DELETE | `/pedidos/{id}`            | Exclui pedido                | 204     |
| GET    | `/eventos`                 | Stream SSE de status         | 200     |
| POST   | `/promocoes/interesse`     | Cadastra interesse           | 201     |
| DELETE | `/promocoes/interesse`     | Cancela interesse            | 200     |

---

### GET `/produtos`

```bash
curl http://localhost:8080/api/produtos
```

Resposta `200` (formato definido pelo MS Estoque; exemplo ilustrativo):

```json
[
  { "id": 1, "nome": "Notebook", "preco": 3500.0, "quantidade": 10 },
  { "id": 2, "nome": "Mouse", "preco": 80.0, "quantidade": 50 }
]
```

---

### POST `/pedidos`

Envio:

```json
{
  "itens": [
    { "produto_id": 1, "quantidade": 2 },
    { "produto_id": 2, "quantidade": 1 }
  ]
}
```

```bash
curl -X POST http://localhost:8080/api/pedidos \
  -H "Content-Type: application/json" \
  -d '{"itens":[{"produto_id":1,"quantidade":2},{"produto_id":2,"quantidade":1}]}'
```

Resposta `201`:

```json
{ "id": 12, "situacao": "CRIADO" }
```

Erros `400`: sem itens, JSON inválido, `quantidade` < 1, produto inexistente ou item duplicado.

```json
{ "erro": "pedido sem itens" }
```

---

### GET `/pedidos`

```bash
# todos
curl http://localhost:8080/api/pedidos

# por situação
curl "http://localhost:8080/api/pedidos?situacao=ENVIADO"
```

Situações: `CRIADO`, `ESTOQUE_DISPONIVEL`, `ESTOQUE_INDISPONIVEL`, `PAGAMENTO_APROVADO`, `PAGAMENTO_RECUSADO`, `ENVIADO`.

Resposta `200` (exemplo ilustrativo):

```json
[
  { "id": 12, "situacao": "ENVIADO" },
  { "id": 13, "situacao": "CRIADO" }
]
```

Situação inválida → `400`:

```json
{ "erro": "situação inválida" }
```

---

### GET `/pedidos/{id}`

```bash
curl http://localhost:8080/api/pedidos/12
```

Resposta `200` (exemplo ilustrativo):

```json
{ "id": 12, "situacao": "ENVIADO" }
```

Erros: `400` (id inválido) e `404`:

```json
{ "erro": "pedido não encontrado" }
```

---

### DELETE `/pedidos/{id}`

```bash
curl -X DELETE http://localhost:8080/api/pedidos/12 -i
```

Resposta `204` (sem corpo). Erros: `400` (id inválido) e `404` (não encontrado).

---

### GET `/eventos` (SSE)

```bash
curl -N http://localhost:8080/api/eventos
```

Cada mudança de status chega como:

```
event:message
data:{"tipo":"pagamento.aprovado","pedido_id":12,"situacao":"PAGAMENTO_APROVADO"}
```

| `tipo`                 | `situacao`             |
|------------------------|------------------------|
| `pedido.estoque_ok`    | `ESTOQUE_DISPONIVEL`   |
| `estoque.indisponivel` | `ESTOQUE_INDISPONIVEL` |
| `pagamento.aprovado`   | `PAGAMENTO_APROVADO`   |
| `pagamento.recusado`   | `PAGAMENTO_RECUSADO`   |
| `pedido.enviado`       | `ENVIADO`              |

- Todos os clientes conectados recebem todos os eventos.
- Eventos enviados sem ninguém conectado são perdidos: ao abrir, consulte `GET /pedidos/{id}`.
- Linhas `: conectado` e `: ping` são comentários SSE (só aparecem no curl).

Exemplo no navegador:

```js
const es = new EventSource("http://localhost:8080/api/eventos");
es.onmessage = (e) => console.log(JSON.parse(e.data));
```

---

### POST `/promocoes/interesse`

Categorias válidas: `A`, `B`, `C`.

Envio:

```json
{
  "email": "seu-email-do-resend@exemplo.com",
  "categorias": ["A", "C"]
}
```

```bash
curl -X POST http://localhost:8080/api/promocoes/interesse \
  -H "Content-Type: application/json" \
  -d '{"email":"seu-email-do-resend@exemplo.com","categorias":["A","C"]}'
```

Resposta `201`:

```json
{ "mensagem": "Solicitação de interesse enviada" }
```

Erro `400` (e-mail ausente/inválido ou JSON inválido). O processamento é assíncrono; categorias fora de A/B/C são ignoradas pelo MS Promoções.

---

### DELETE `/promocoes/interesse`

```bash
curl -X DELETE "http://localhost:8080/api/promocoes/interesse?email=seu-email-do-resend@exemplo.com"
```

Resposta `200`:

```json
{ "mensagem": "Solicitação de cancelamento enviada" }
```

Erro `400` se faltar o `email`:

```json
{ "erro": "informe o e-mail via ?email=" }
```

---

## Dica (Windows)

No CMD/PowerShell as aspas simples não funcionam no `curl`. Salve o JSON em um arquivo e use:

```bash
curl -X POST http://localhost:8080/api/pedidos -H "Content-Type: application/json" -d @pedido.json
```
