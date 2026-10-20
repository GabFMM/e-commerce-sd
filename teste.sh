#!/usr/bin/env bash
# =============================================================================
# Script de testes da API do E-commerce (MS Principal / API Gateway)
#
# Uso:
#   ./testar_api.sh                 # testes automáticos
#   E2E=1 ./testar_api.sh           # + acompanha um pedido até o fim (você
#                                   #   aprova/recusa o pagamento no Mock)
#
# Requisitos: bash, curl. (No Windows: Git Bash ou WSL.)
# Suba antes: RabbitMQ + MS Principal (+ Estoque, para os testes de SSE).
#
# Variáveis opcionais:
#   BASE_URL      URL base da API            (padrão: http://localhost:8080/api)
#   PRODUTO_ID    id de produto existente    (padrão: 1)
#   PRODUTO_ID_2  outro id de produto        (padrão: 2)
#   ORIGIN        origem do frontend p/ CORS (padrão: http://localhost:5173)
#   ESTOQUE_URL   URL do MS Estoque          (padrão: http://localhost:8081)
#   SSE_WAIT      segundos esperando evento  (padrão: 15)
#   E2E           1 para ligar o teste ponta a ponta (padrão: 0)
#   E2E_TIMEOUT   segundos do teste E2E      (padrão: 120)
#
# Atenção: o script cria 3 pedidos de teste (consome um pouco de estoque).
# =============================================================================
PAGAMENTO_URL="${PAGAMENTO_URL:-http://localhost:8083}"
BASE_URL="${BASE_URL:-http://localhost:8080/api}"
PRODUTO_ID="${PRODUTO_ID:-1}"
PRODUTO_ID_2="${PRODUTO_ID_2:-2}"
ORIGIN="${ORIGIN:-http://localhost:5173}"
ESTOQUE_URL="${ESTOQUE_URL:-http://localhost:8081}"
SSE_WAIT="${SSE_WAIT:-15}"
E2E="${E2E:-0}"
E2E_TIMEOUT="${E2E_TIMEOUT:-120}"

# ---------- cores (só se for terminal) ----------
if [ -t 1 ]; then
  G=$'\e[32m'; R=$'\e[31m'; Y=$'\e[33m'; B=$'\e[1m'; N=$'\e[0m'
else
  G=""; R=""; Y=""; B=""; N=""
fi

PASS=0
FAIL=0
STATUS=""
BODY=""
PIDS=()
TMP="$(mktemp -d)"

cleanup() {
  for p in "${PIDS[@]}"; do kill "$p" 2>/dev/null; done
  rm -rf "$TMP"
}
trap cleanup EXIT

# ---------- helpers ----------
secao() { printf "\n%s== %s ==%s\n" "$B" "$1" "$N"; }
ok()    { PASS=$((PASS+1)); printf "  %s[OK]%s   %s\n" "$G" "$N" "$1"; }
fail()  { FAIL=$((FAIL+1)); printf "  %s[FALHA]%s %s\n" "$R" "$N" "$1"; }
info()  { printf "  %s[info]%s %s\n" "$Y" "$N" "$1"; }

# req METODO CAMINHO [CORPO_JSON]  -> preenche STATUS e BODY
req() {
  local method="$1" path="$2" data="${3-}" out
  if [ -n "$data" ]; then
    out=$(curl -s -m 10 -w $'\n%{http_code}' -X "$method" \
      -H "Content-Type: application/json" -d "$data" "$BASE_URL$path")
  else
    out=$(curl -s -m 10 -w $'\n%{http_code}' -X "$method" "$BASE_URL$path")
  fi
  STATUS="${out##*$'\n'}"
  BODY="${out%$'\n'*}"
}

# check_status "descrição" ESPERADO [ESPERADO2 ...]
check_status() {
  local desc="$1"; shift
  local esperado
  for esperado in "$@"; do
    if [ "$STATUS" = "$esperado" ]; then
      ok "$desc (HTTP $STATUS)"
      return 0
    fi
  done
  fail "$desc — esperado HTTP $*, recebido $STATUS | corpo: ${BODY:0:160}"
  return 1
}

# check_body_contains "descrição" "trecho"
check_body_contains() {
  local compacto="${BODY// /}"   # tolera espaços no JSON
  if [[ "$compacto" == *"$2"* ]]; then
    ok "$1"
  else
    fail "$1 — corpo não contém '$2' | corpo: ${BODY:0:160}"
  fi
}

# extrai o campo "id" numérico do corpo da resposta
extrair_id() { printf "%s" "$BODY" | tr -d ' ' | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p'; }

# aguardar_evento ARQUIVO PEDIDO_ID TIMEOUT_SEG
aguardar_evento() {
  local t=0
  while [ "$t" -lt "$3" ]; do
    grep -Eq "\"pedido_id\":$2[,}]" "$1" 2>/dev/null && return 0
    sleep 1
    t=$((t+1))
  done
  return 1
}

# =============================================================================
secao "0. Pré-checagem"
# =============================================================================
req GET /produtos
if [ "$STATUS" = "000" ]; then
  printf "%s[ERRO]%s Não consegui conectar em %s. O MS Principal está rodando?\n" "$R" "$N" "$BASE_URL"
  exit 2
fi
ok "MS Principal respondendo em $BASE_URL"

# =============================================================================
secao "1. CORS (preflight do navegador)"
# =============================================================================
HDRS=$(curl -s -m 10 -D - -o /dev/null -X OPTIONS "$BASE_URL/pedidos" \
  -H "Origin: $ORIGIN" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: content-type" | tr -d '\r')

PF_STATUS=$(printf "%s" "$HDRS" | head -1 | awk '{print $2}')
if [ "$PF_STATUS" = "204" ] || [ "$PF_STATUS" = "200" ]; then
  ok "Preflight OPTIONS /pedidos (HTTP $PF_STATUS)"
else
  fail "Preflight OPTIONS /pedidos — HTTP $PF_STATUS"
fi

if printf "%s" "$HDRS" | grep -qi "^access-control-allow-origin: *$ORIGIN"; then
  ok "Access-Control-Allow-Origin liberou $ORIGIN"
else
  fail "Access-Control-Allow-Origin não liberou $ORIGIN (ajuste AllowOrigins no main.go)"
fi

if printf "%s" "$HDRS" | grep -i "^access-control-allow-methods" | grep -qi "DELETE"; then
  ok "Access-Control-Allow-Methods inclui DELETE"
else
  fail "Access-Control-Allow-Methods não inclui DELETE"
fi

# =============================================================================
secao "2. Produtos"
# =============================================================================
req GET /produtos
check_status "GET /produtos" 200
if [[ "$BODY" == \[* ]]; then ok "Corpo é uma lista JSON"; else fail "Corpo não é uma lista JSON: ${BODY:0:100}"; fi

secao "2b. MS Estoque (REST direto)"
out=$(curl -s -m 10 -w $'\n%{http_code}' "$ESTOQUE_URL/produtos")
STATUS="${out##*$'\n'}"
BODY="${out%$'\n'*}"
check_status "GET $ESTOQUE_URL/produtos" 200
if [[ "$BODY" == \[* ]]; then ok "Estoque devolve uma lista JSON"; else fail "Estoque não devolveu lista JSON: ${BODY:0:100}"; fi

# =============================================================================
secao "3. Criação de pedidos: casos inválidos"
# =============================================================================
req POST /pedidos '{"itens":[]}'
check_status "Pedido sem itens" 400

req POST /pedidos '{}'
check_status "Corpo sem o campo itens" 400

req POST /pedidos '{"itens":'
check_status "JSON malformado" 400

req POST /pedidos "{\"itens\":[{\"produto_id\":$PRODUTO_ID,\"quantidade\":0}]}"
check_status "Quantidade 0" 400

req POST /pedidos "{\"itens\":[{\"produto_id\":$PRODUTO_ID,\"quantidade\":-1}]}"
check_status "Quantidade negativa" 400

req POST /pedidos '{"itens":[{"produto_id":999999,"quantidade":1}]}'
check_status "Produto inexistente" 400

req POST /pedidos "{\"itens\":[{\"produto_id\":$PRODUTO_ID,\"quantidade\":1},{\"produto_id\":$PRODUTO_ID,\"quantidade\":1}]}"
check_status "Item duplicado no mesmo pedido" 400

# =============================================================================
secao "4. Criação e consulta de pedidos"
# =============================================================================
req POST /pedidos "{\"itens\":[{\"produto_id\":$PRODUTO_ID,\"quantidade\":1},{\"produto_id\":$PRODUTO_ID_2,\"quantidade\":1}]}"
check_status "POST /pedidos (2 itens)" 201
check_body_contains "Situação inicial é CRIADO" '"situacao":"CRIADO"'
PEDIDO_ID="$(extrair_id)"

if [ -z "$PEDIDO_ID" ]; then
  fail "Não consegui extrair o id do pedido criado (resposta: ${BODY:0:100})"
else
  ok "Pedido criado com id $PEDIDO_ID"

  req GET "/pedidos/$PEDIDO_ID"
  check_status "GET /pedidos/$PEDIDO_ID" 200
fi

req GET /pedidos
check_status "GET /pedidos" 200
if [[ "$BODY" == \[* ]]; then ok "Listagem é uma lista JSON"; else fail "Listagem não é lista JSON: ${BODY:0:100}"; fi

req GET "/pedidos?situacao=CRIADO"
check_status "GET /pedidos?situacao=CRIADO" 200

req GET "/pedidos?situacao=ENVIADO"
check_status "GET /pedidos?situacao=ENVIADO" 200

req GET "/pedidos?situacao=INVALIDA"
check_status "Filtro com situação inválida" 400

req GET /pedidos/abc
check_status "GET /pedidos/abc (id não numérico)" 400

req GET /pedidos/999999
check_status "GET /pedidos/999999 (inexistente)" 404

# =============================================================================
secao "5. SSE (hub com broadcast)"
# =============================================================================
if [ "$E2E" = "1" ]; then SSE_MAX=$((E2E_TIMEOUT + 15)); else SSE_MAX=$((SSE_WAIT + 15)); fi

# Dois clientes conectados ao mesmo tempo: ambos devem receber o mesmo evento
curl -s -N -m "$SSE_MAX" -H "Origin: $ORIGIN" "$BASE_URL/eventos" > "$TMP/sse_a.txt" 2>/dev/null &
PIDS+=("$!")
curl -s -N -m "$SSE_MAX" -H "Origin: $ORIGIN" "$BASE_URL/eventos" > "$TMP/sse_b.txt" 2>/dev/null &
PIDS+=("$!")
sleep 1.5

if grep -q "^: conectado" "$TMP/sse_a.txt"; then
  ok "Cliente A recebeu o flush inicial (': conectado')"
else
  fail "Cliente A não recebeu o flush inicial (headers não foram enviados na conexão?)"
fi

req POST /pedidos "{\"itens\":[{\"produto_id\":$PRODUTO_ID,\"quantidade\":1}]}"
check_status "Cria pedido para disparar eventos" 201
SSE_PEDIDO="$(extrair_id)"

if [ -n "$SSE_PEDIDO" ]; then
  info "Aguardando até ${SSE_WAIT}s por eventos do pedido $SSE_PEDIDO (precisa do MS Estoque rodando)..."

  if aguardar_evento "$TMP/sse_a.txt" "$SSE_PEDIDO" "$SSE_WAIT"; then
    ok "Cliente A recebeu evento do pedido $SSE_PEDIDO"
  else
    fail "Cliente A não recebeu evento do pedido $SSE_PEDIDO em ${SSE_WAIT}s (MS Estoque/RabbitMQ no ar? filas configuradas?)"
  fi

  if aguardar_evento "$TMP/sse_b.txt" "$SSE_PEDIDO" 3; then
    ok "Cliente B também recebeu o evento (broadcast funcionando)"
  else
    fail "Cliente B NÃO recebeu o evento (o evento foi para um cliente só?)"
  fi

  if grep -E "\"pedido_id\":$SSE_PEDIDO[,}]" "$TMP/sse_a.txt" | grep -q '"tipo":"[a-z_.]*"'; then
    ok "Formato do evento contém tipo, pedido_id e situacao"
    grep -E "\"pedido_id\":$SSE_PEDIDO[,}]" "$TMP/sse_a.txt" | sed 's/^/        /'
  else
    fail "Formato do evento inesperado"
  fi
fi

# ---------- teste ponta a ponta opcional ----------
if [ "$E2E" = "1" ] && [ -n "$SSE_PEDIDO" ]; then
  secao "5b. Ponta a ponta (E2E)"
  info "Abra a URL do Mock de Pagamento do pedido $SSE_PEDIDO e clique em Aprovado ou Recusado."
  info "Aguardando até ${E2E_TIMEOUT}s pelo desfecho do pedido..."

  FINAL='"tipo":"(pedido\.enviado|pagamento\.recusado|estoque\.indisponivel)","pedido_id":'"$SSE_PEDIDO"'[,}]'
  t=0
  while [ "$t" -lt "$E2E_TIMEOUT" ]; do
    grep -Eq "$FINAL" "$TMP/sse_a.txt" && break
    sleep 1
    t=$((t+1))
  done

  echo "      Eventos recebidos para o pedido $SSE_PEDIDO:"
  grep -E "\"pedido_id\":$SSE_PEDIDO[,}]" "$TMP/sse_a.txt" | sed 's/^/        /'

  if grep -Eq "$FINAL" "$TMP/sse_a.txt"; then
    ok "Pedido $SSE_PEDIDO chegou a um estado final via SSE"
  else
    fail "Pedido $SSE_PEDIDO não chegou a estado final em ${E2E_TIMEOUT}s"
  fi
fi

# encerra os clientes SSE
for p in "${PIDS[@]}"; do kill "$p" 2>/dev/null; done
PIDS=()

# =============================================================================
secao "5c. Pagamento (webhook + Mock)"
# =============================================================================
webhook() {
  local out
  out=$(curl -s -m 10 -w $'\n%{http_code}' -X POST \
    -H "Content-Type: application/json" -d "$1" "$PAGAMENTO_URL/webhook/pagamento")
  STATUS="${out##*$'\n'}"
  BODY="${out%$'\n'*}"
}

# Casos inválidos (não dependem do Mock)
webhook '{"pedido_id":1}'
check_status "Webhook sem status" 400
webhook '{"pedido_id":1,"status":"TALVEZ"}'
check_status "Webhook com status inválido" 400
webhook '{"pedido_id":999999,"status":"APROVADO"}'
check_status "Webhook de pedido sem cobrança pendente" 404

if [ -n "$SSE_PEDIDO" ]; then
  CRIADO="\"tipo\":\"pagamento.criado\",\"pedido_id\":$SSE_PEDIDO,"
  t=0
  while [ "$t" -lt 10 ] && ! grep -Fq "$CRIADO" "$TMP/sse_a.txt"; do sleep 1; t=$((t+1)); done

  if grep -Fq "$CRIADO" "$TMP/sse_a.txt"; then
    URL=$(grep -F "$CRIADO" "$TMP/sse_a.txt" | sed -n 's/.*"url_checkout":"\([^"]*\)".*/\1/p' | head -1)
    ok "Evento pagamento.criado chegou com url_checkout"
    info "URL do checkout: $URL"

    # Sem E2E, o script faz o papel do Mock (clique em "Aprovado")
    if [ "$E2E" != "1" ]; then
      webhook "{\"pedido_id\":$SSE_PEDIDO,\"status\":\"APROVADO\"}"
      check_status "Webhook APROVADO (simulando o clique no Mock)" 200

      webhook "{\"pedido_id\":$SSE_PEDIDO,\"status\":\"APROVADO\"}"
      check_status "Webhook repetido é rejeitado" 404

      ENVIADO="\"tipo\":\"pedido.enviado\",\"pedido_id\":$SSE_PEDIDO,"
      t=0
      while [ "$t" -lt 10 ] && ! grep -Fq "$ENVIADO" "$TMP/sse_a.txt"; do sleep 1; t=$((t+1)); done
      if grep -Fq "$ENVIADO" "$TMP/sse_a.txt"; then
        ok "Pedido $SSE_PEDIDO chegou a pedido.enviado após o pagamento aprovado"
      else
        fail "pedido.enviado não chegou em 10s (MS Entrega no ar?)"
      fi
    fi
  else
    fail "pagamento.criado não chegou em 10s (Mock no ar? Sem o Mock o esperado é pagamento.recusado)"
  fi
fi
# =============================================================================
secao "6. Exclusão de pedidos"
# =============================================================================
req POST /pedidos "{\"itens\":[{\"produto_id\":$PRODUTO_ID,\"quantidade\":1}]}"
check_status "Cria pedido para excluir" 201
DEL_ID="$(extrair_id)"

if [ -n "$DEL_ID" ]; then
  req DELETE "/pedidos/$DEL_ID"
  check_status "DELETE /pedidos/$DEL_ID" 204

  req GET "/pedidos/$DEL_ID"
  check_status "Pedido excluído some da consulta" 404

  req DELETE "/pedidos/$DEL_ID"
  check_status "Excluir de novo retorna não encontrado" 404
fi

req DELETE /pedidos/abc
check_status "DELETE /pedidos/abc (id não numérico)" 400

req DELETE /pedidos/999999
check_status "DELETE /pedidos/999999 (inexistente)" 404

# =============================================================================
secao "7. Promoções (interesse)"
# =============================================================================
EMAIL="yagoribeiro@alunos.utfpr.edu.br"

req POST /promocoes/interesse "{\"email\":\"$EMAIL\",\"categorias\":[\"A\",\"C\"]}"
check_status "Cadastrar interesse válido" 201

req POST /promocoes/interesse "{\"email\":\"$EMAIL\",\"categorias\":[\"b\"]}"
check_status "Acrescentar categoria em minúscula ao mesmo e-mail" 201

req POST /promocoes/interesse '{"email":"nao-e-um-email","categorias":["A"]}'
check_status "E-mail inválido" 400

req POST /promocoes/interesse '{"categorias":["A"]}'
check_status "Sem e-mail" 400

req POST /promocoes/interesse '{"email":"x'"$EMAIL"'","categorias":["Z"]}'
check_status "Categoria inválida (201 sem validação no gateway / 400 com validação)" 201 400

req POST /promocoes/interesse '{"email":"y'"$EMAIL"'","categorias":[]}'
check_status "Categorias vazias (201 sem validação no gateway / 400 com validação)" 201 400

req DELETE "/promocoes/interesse?email=$EMAIL"
check_status "Cancelar interesse" 200

req DELETE /promocoes/interesse
check_status "Cancelar sem informar e-mail" 400

info "Confira no log do MS Promoções as linhas [INFO-PROMOCOES] 'Interesse registrado' e 'Interesse removido' para $EMAIL"

# =============================================================================
secao "Resumo"
# =============================================================================
TOTAL=$((PASS+FAIL))
printf "  Total: %d | %sOK: %d%s | %sFalhas: %d%s\n\n" "$TOTAL" "$G" "$PASS" "$N" "$R" "$FAIL" "$N"

[ "$FAIL" -eq 0 ] && exit 0 || exit 1
