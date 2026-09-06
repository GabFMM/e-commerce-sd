# E-commerce SD

## Como executar

1. Execute o RabbitMQ
```
docker run -d \
  --hostname rabbitmq \
  --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  -v rabbitmq_data:/var/lib/rabbitmq \
  rabbitmq:4-management
```

2. Crie as chaves privadas e públicas para cada microsserviço
```
go run ./seguranca/gerar-chaves
```

3. Configure as filas do RabbitMQ
```
go run ./setup
```