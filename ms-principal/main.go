package main

import (
	"context"
	"crypto/ed25519"
	"e-commerce-sd/ms-principal/dto"
	"e-commerce-sd/ms-principal/service"
	"e-commerce-sd/seguranca"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeExchange = "eCommerce"
const NomeServico = "principal"
const NomeFila = "principal"

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
	failOnError(err, "[ERRO-MS-PRINCIPAL] Erro ao conectar ao RabbitMQ")
	defer conn.Close()

	// Canal dedicado para publicar e canal dedicado para consumir
	publishCh, err := conn.Channel()
	failOnError(err, "[ERRO-MS-PRINCIPAL] Erro ao abrir channel de publish")
	defer publishCh.Close()

	consumeCh, err := conn.Channel()
	failOnError(err, "[ERRO-MS-PRINCIPAL] Erro ao abrir channel de consume")
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
	failOnError(err, "[ERRO-MS-PRINCIPAL] Falha ao registrar na fila do consumidor")

	chavesPublicas, err := seguranca.CarregarTodasChavesPublicas("ms-estoque/")
	failOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível carregar as chaves públicas")

	chavePrivada, err := seguranca.CarregarChavePrivada("chaves/estoque_private.pem")
	failOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível carregar chave privada")

	var forever chan struct{}

	go func() {
		for d := range msgs {
			payload, err := seguranca.AbrirPacote(d.Body, chavesPublicas)

			printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível descriptografar/autenticar/interpretar pacote recebido")
			if err != nil {
				continue
			}

			switch d.RoutingKey {
			case "pagamento.aprovado":
				pagamentoAprovado(payload)
			case "pagamento.recusado":
				pagamentoRecusado(publishCh, chavePrivada, payload)
			case "pedido.enviado":
				pedidoEnviado(payload)
			case "pedido.estoque_ok":
				estoqueOk(payload)
			case "estoque.indisponivel":
				estoqueIndisponivel(publishCh, chavePrivada, payload)
			default:
				printOnError(err, "[ERRO-MS-PRINCIPAL] routing key inválida:"+d.RoutingKey)
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func pagamentoAprovado(payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "PAGAMENTO_APROVADO")
	// Não publica nada
}

func pagamentoRecusado(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "PAGAMENTO_RECUSADO")
	
	publicar(publishCh, chavePrivada, "pedido.excluido", pedidoDTO)
}

func pedidoEnviado(payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "ENVIADO")
	// Não publica nada
}

func estoqueOk(payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "ESTOQUE_DISPONIVEL")
	// Não publica nada
}

func estoqueIndisponivel(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "ESTOQUE_INDISPONIVEL")
	
	publicar(publishCh, chavePrivada, "pedido.excluido", pedidoDTO)
}

func publicar(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, routingKey string, body interface{}) {
	bodyByte, err := seguranca.CriarPacote(body, NomeServico, chavePrivada)
	if err != nil {
		log.Printf("[ERRO-MS-PRINCIPAL] Não foi possível criar o pacote para %s: %v", routingKey, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = publishCh.PublishWithContext(ctx,
		NomeExchange,        // exchange
		routingKey,          // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        bodyByte,
		},
	)

	if err != nil {
		log.Printf("[ERRO-MS-PRINCIPAL] Falha ao publicar pedido.estoque_ok: %v", err)
		return
	}

	log.Printf("[INFO-MS-PRINCIPAL] Pacote publicado para %s", routingKey)
}