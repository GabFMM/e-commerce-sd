# E-commerce SD

## Como executar

1. Configure o arquivo *.env*

2. Execute o RabbitMQ
```
docker run -d \
  --hostname rabbitmq \
  --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  -v rabbitmq_data:/var/lib/rabbitmq \
  rabbitmq:4-management
```

3. Prepare o ambiente com as configurações necessárias
```
go run ./setup/main iniciar
```

4. Execute o MS Principal
```
go run ./ms-principal
```

5. Execute o front-end
```
cd frontend
npm install
ng serve
```

5. Após utilizar o sistema de e-commerce, remova os dados do banco
```
go run ./setup/main encerrar
```