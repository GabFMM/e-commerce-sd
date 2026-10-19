# E-commerce SD

## Como executar

### 1. Configurar as variáveis de ambiente

Configure o arquivo `.env.example` conforme necessário e renomeie-o para `.env`.

### 2. Executar o projeto completo

Na raiz do projeto, execute:

```bash
make dev
```

Esse comando inicia os containers Docker, configura as filas do RabbitMQ e cria as chaves necessárias, instala as dependências do frontend e inicia os microsserviços e o frontend em paralelo.

Para encerrar os serviços iniciados pelo script, pressione `Ctrl+C`.

### 3. Executar os componentes separadamente

Caso prefira executar os componentes individualmente, utilize os comandos abaixo em terminais separados.

**Infraestrutura e configuração do RabbitMQ:**

```bash
make setup
```

**Microsserviços:**

```bash
make principal
make estoque
make pagamento
make entrega
make promocoes
```

**Frontend:**

```bash
make frontend
```

O comando `make frontend` instala as dependências do frontend e inicia o servidor de desenvolvimento Angular.

### 4. Comandos disponíveis

| Comando | Descrição |
|---|---|
| `make setup` | Inicia os containers e configura o RabbitMQ |
| `make docker` | Inicia os containers Docker |
| `make setup-rabbitmq` | Configura as filas do RabbitMQ e cria as chaves necessárias |
| `make principal` | Inicia o microsserviço principal |
| `make estoque` | Inicia o microsserviço de estoque |
| `make pagamento` | Inicia o microsserviço de pagamento |
| `make entrega` | Inicia o microsserviço de entrega |
| `make promocoes` | Inicia o microsserviço de promoções |
| `make frontend` | Instala as dependências e inicia o frontend |
| `make dev` | Inicia o projeto completo |

### Observações

- Execute os comandos a partir da raiz do projeto.
- O Docker e o Docker Compose, o Go, o Node.js e o npm devem estar instalados.
- O comando `make dev` utiliza o script `scripts/dev.sh` para gerenciar a execução paralela dos serviços.
- Os comandos individuais permitem acompanhar e depurar cada componente separadamente.
- A configuração do RabbitMQ depende de o serviço estar disponível para receber conexões.