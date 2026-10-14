package main

import (
	"context"
	"crypto/ed25519"
	"e-commerce-sd/ms-estoque/dto"
	"e-commerce-sd/ms-estoque/service"
	"e-commerce-sd/seguranca"
	"encoding/json"
	"log"
	"time"

	"net/http"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/gin-gonic/gin"
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

func montarRouterEstoque() *gin.Engine {
	r := gin.Default()

	// GET /produtos -> produtos com saldo disponível (consumido pelo MS Principal)
	r.GET("/produtos", func(c *gin.Context) {
		produtos, err := service.ListarProdutosDisponiveis()
		if err != nil {
			log.Printf("[ERRO-MS-ESTOQUE] Falha ao consultar produtos: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "não foi possível consultar os produtos"})
			return
		}
		c.JSON(http.StatusOK, produtos)
	})

	return r
}

// Sobe a API REST em uma goroutine. Endereço padrão ":8081" (troque com ESTOQUE_ADDR).
func iniciarAPIEstoque() {
	addr := os.Getenv("ESTOQUE_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: montarRouterEstoque(),
	}

	go func() {
		log.Printf("[INFO-MS-ESTOQUE] API REST escutando em %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Panicf("[ERRO-MS-ESTOQUE] Erro no servidor HTTP: %s", err)
		}
	}()
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

	chavesPublicas, err := seguranca.CarregarTodasChavesPublicas("ms-estoque/chaves/")
	failOnError(err, "[ERRO-MS-ESTOQUE] Não foi possível carregar as chaves públicas")

	chavePrivada, err := seguranca.CarregarChavePrivada("ms-estoque/chaves/estoque_private.pem")
	failOnError(err, "[ERRO-MS-ESTOQUE] Não foi possível carregar chave privada")

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
				verificarDisponibilidade(publishCh, chavePrivada, payload)
			case "pedido.excluido":
				removerReservas(payload)
			default:
				log.Printf("[ERRO-MS-ESTOQUE] routing key inválida: %s", d.RoutingKey)
			}
		}
	}()

	// API REST (GET /produtos) consumida pelo MS Principal
	iniciarAPIEstoque()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func verificarDisponibilidade(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-ESTOQUE] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	if service.VerificarDisponibilidade(pedidoDTO) {
		service.ReservarProdutos(pedidoDTO)
		publicar(publishCh, chavePrivada, "pedido.estoque_ok", pedidoDTO)
	} else {
		publicar(publishCh, chavePrivada, "estoque.indisponivel", pedidoDTO)
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
	// Não publica nada, conforme requisitos
}

func publicar(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, routingKey string, body interface{}) {
	bodyByte, err := seguranca.CriarPacote(body, NomeServico, chavePrivada)
	if err != nil {
		log.Printf("[ERRO-MS-ESTOQUE] Não foi possível criar o pacote para %s: %v", routingKey, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = publishCh.PublishWithContext(ctx,
		NomeExchange, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        bodyByte,
		},
	)

	if err != nil {
		log.Printf("[ERRO-MS-ESTOQUE] Falha ao publicar %s: %v", routingKey, err)
		return
	}

	log.Printf("[INFO-MS-ESTOQUE] Pacote publicado para %s", routingKey)
}
