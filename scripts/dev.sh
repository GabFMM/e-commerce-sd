#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

pids=()

# Encerra os grupos de processos iniciados pelo script.
cleanup() {
    trap - EXIT INT TERM

    echo
    echo "Encerrando os serviços..."

    for pid in "${pids[@]}"; do
        kill -TERM -- "-$pid" 2>/dev/null || true
    done

    for pid in "${pids[@]}"; do
        wait "$pid" 2>/dev/null || true
    done

    echo "Serviços encerrados."
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

echo "Iniciando Docker..."
docker compose --env-file .env -f docker/docker-compose.yml up -d

echo "Configurando filas no RabbitMQ e criando chaves..."
go run ./setup/main

echo "Instalando dependências do frontend..."
npm --prefix frontend install

# Inicia um comando em uma sessão própria, permitindo encerrar
# o grupo de processos correspondente posteriormente.
start_service() {
    local name="$1"
    shift

    echo "Iniciando $name..."
    setsid "$@" &
    pids+=("$!")
}

start_service "ms-principal" go run ./ms-principal
start_service "ms-estoque" go run ./ms-estoque
start_service "ms-pagamento" go run ./ms-pagamento
start_service "ms-entrega" go run ./ms-entrega
start_service "ms-promocoes" go run ./ms-promocoes

start_service "frontend" bash -c \
    'cd frontend && exec npx ng serve'

echo
echo "Projeto iniciado. Pressione Ctrl+C para encerrar os serviços."
echo

# Aguarda o primeiro processo terminar.
# Se um serviço parar ou falhar, o script termina e o trap
# encerra os demais.
wait -n "${pids[@]}"