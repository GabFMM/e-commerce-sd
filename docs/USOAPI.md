# Uso da API

API do **MS Principal**. Pedidos e produtos usam GraphQL; SSE e promoções continuam com endpoints HTTP.

- **GraphQL:** `http://localhost:8080/api/graphql`
- **SSE e promoções:** base URL `http://localhost:8080/api`
- **Content-Type do GraphQL:** `application/json`

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

## Operações GraphQL

O endpoint aceita `POST` com um JSON que contém `query` e, opcionalmente, `variables`.

### Consultar produtos e pedidos

```bash
curl -X POST http://localhost:8080/api/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"query { produtos { id nome categoria quantidadeDisponivel } pedidos { id situacao produtos { produtoId quantidade } } }"}'
```

### Criar pedido

```bash
curl -X POST http://localhost:8080/api/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { criarPedido(request: { itens: [{ produtoId: 1, quantidade: 2 }, { produtoId: 2, quantidade: 1 }] }) }"}'
```

Resposta:

```json
{"data":{"criarPedido":true}}
```

### Remover pedido

```bash
curl -X POST http://localhost:8080/api/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { removerPedido(request: { id: 12 }) }"}'
```

Erros de validação e execução aparecem no campo GraphQL `errors`.

## Endpoints HTTP

| Método | Rota | Descrição |
|---|---|---|
| GET | `/api/eventos` | Stream SSE de status |
| POST | `/api/promocoes/interesse` | Cadastra interesse |
| DELETE | `/api/promocoes/interesse` | Cancela interesse |

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
- Eventos enviados sem ninguém conectado são perdidos: ao abrir ou reconectar, o frontend pode atualizar a lista com a query GraphQL `pedidos`.
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

No CMD/PowerShell as aspas simples não funcionam no `curl`. Salve o corpo JSON da operação GraphQL em um arquivo e use:

```bash
curl -X POST http://localhost:8080/api/graphql -H "Content-Type: application/json" -d @consulta.json
```
