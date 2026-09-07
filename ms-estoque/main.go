package main

import (
	"e-commerce-sd/ms-estoque/dto"
	"e-commerce-sd/ms-estoque/service"
	"e-commerce-sd/seguranca"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeExchange = "eCommerce"
const NomeServico = "estoque"
const NomeFila = "estoque"

// Encerra o MS
func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

// Não encerra o MS
func printOnError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "[ERRO-MS-ESTOQUE] Falha ao conectar no RabbitMQ")
	defer conn.Close()

	// Canal dedicado para publicar e canal dedicado para consumir
	publishCh, err := conn.Channel()
	failOnError(err, "[ERRO-MS-ESTOQUE] Erro ao abrir channel de publish")
	defer publishCh.Close()

	consumeCh, err := conn.Channel()
	failOnError(err, "[ERRO-MS-ESTOQUE] Erro ao abrir channel de consume")
	defer consumeCh.Close()

	// Liga-se na fila de consumidores
	msgs, err := consumeCh.Consume(
		NomeFila, // queue
		"",       // id do consumidor (vazio para o rabbitmq gerar automaticamente)
		true,     // auto-ack
		true,     // exclusive (apenas o ms-estoque lê a fila dele)
		false,    // no-local
		false,    // no-wait
		nil,      // args
	)
	failOnError(err, "[ERRO-MS-ESTOQUE] Falha ao registrar na fila do consumidor")

	chavesPublicas, err := seguranca.CarregarTodasChavesPublicas("ms-estoque/")
	failOnError(err, "[ERRO-MS-ESTOQUE] Não foi possível carregar as chaves públicas")

	var forever chan struct{}

	go func() {
		for d := range msgs {
			payload, err := seguranca.AbrirPacote(d.Body, chavesPublicas)

			printOnError(err, "[ERRO-MS-ESTOQUE] Não foi possível descriptografar/autenticar/interpretar pacote recebido")
			if err != nil {
				continue
			}

			switch d.RoutingKey {
			case "pedido.criado":
				verificarDisponibilidade(payload)
			case "pedido.excluido":
				removerReservas(payload)
			default:
				printOnError(err, "[ERRO-MS-ESTOQUE] routing key inválida:"+d.RoutingKey)
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func verificarDisponibilidade(payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-ESTOQUE] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	if service.VerificarDisponibilidade(pedidoDTO) {
		service.ReservarProdutos(pedidoDTO)
	} else {
		
	}
}

func removerReservas(payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-ESTOQUE] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.RemoverReservas(pedidoDTO)
}
