.PHONY: setup docker setup-rabbitmq principal estoque pagamento entrega promocoes frontend dev

# Infraestrutura
setup: docker setup-rabbitmq

docker:
	docker compose --env-file .env -f docker/docker-compose.yml up -d

setup-rabbitmq:
	go run ./setup/main

# Microsserviços individuais
principal:
	go run ./ms-principal

estoque:
	go run ./ms-estoque

pagamento:
	go run ./ms-pagamento

entrega:
	go run ./ms-entrega

promocoes:
	go run ./ms-promocoes

# Frontend individual
frontend:
	cd frontend && npm install && npx ng serve

# Executa o projeto completo
dev:
	bash scripts/dev.sh