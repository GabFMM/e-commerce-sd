package main

import (
	"context"
	"crypto/ed25519"
	"e-commerce-sd/seguranca"
	"encoding/json"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeExchange = "eCommerce"
const NomeServico = "entrega"
const NomeFila = "entrega"

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

type PedidoAprovado struct {
	PedidoID int `json:"pedido_id"`
}

// PedidoEnviado é a struct de negócio que vamos publicar.
type PedidoEnviado struct {
	PedidoID int    `json:"pedido_id"`
	NotaID   string `json:"nota_id"`
}

// publicarPedidoEnviado monta o pacote assinado e publica na
// exchange eCommerce com a routing key pedido.enviado.
func publicarPedidoEnviado(ch *amqp.Channel, evento PedidoEnviado, chavePrivada ed25519.PrivateKey) {
	pacoteBytes, err := seguranca.CriarPacote(evento, NomeServico, chavePrivada)
	if err != nil {
		log.Printf(" [!] Falha ao criar pacote assinado: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx,
		NomeExchange,     // exchange
		"pedido.enviado", // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        pacoteBytes,
		},
	)
	if err != nil {
		log.Printf(" [!] Falha ao publicar pedido.enviado: %v", err)
		return
	}

	log.Printf(" [x] Publicado pedido.enviado para pedido %s (nota %s)", evento.PedidoID, evento.NotaID)
}

func handlePagamentoAprovado(payload []byte, publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey) {
	var dados PedidoAprovado
	if err := json.Unmarshal(payload, &dados); err != nil {
		log.Printf(" [ERRO] Falha ao desserializar payload: %v", err)
		return
	}

	log.Printf(" [x] Processando entrega do pedido %s...", dados.PedidoID)

	// Simulação da emissão de nota / preparo de entrega
	time.Sleep(500 * time.Millisecond)
	notaID := "NF-" + string(dados.PedidoID)

	evento := PedidoEnviado{
		PedidoID: dados.PedidoID,
		NotaID:   notaID,
	}

	publicarPedidoEnviado(publishCh, evento, chavePrivada)
}

func consumirEProcessar(
	consumeCh *amqp.Channel,
	publishCh *amqp.Channel,
	chavePrivada ed25519.PrivateKey,
	chavesPublicas map[string]ed25519.PublicKey,
) {
	msgs, err := consumeCh.Consume(
		NomeFila, // queue
		"",       // consumer
		true,     // auto-ack
		false,    // exclusive
		false,    // no-local
		false,    // no-wait
		nil,      // args
	)
	failOnError(err, "Erro ao registrar consumidor na fila "+NomeFila)

	for d := range msgs {
		payload, err := seguranca.AbrirPacote(d.Body, chavesPublicas)
		if err != nil {
			log.Printf(" [ERRO] Evento descartado (%s): %v", d.RoutingKey, err)
			continue
		}

		//Aqui seria quando fazemos o procesamento e o publish de algo
		switch d.RoutingKey {
		case "pagamento.aprovado":
			handlePagamentoAprovado(payload, publishCh, chavePrivada)
		default:
			log.Printf(" [!] Routing key inesperada na fila %s: %s", NomeFila, d.RoutingKey)
		}
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Erro ao conectar ao RabbitMQ")
	defer conn.Close()

	// Canal dedicado para publicar e canal dedicado para consumir
	publishCh, err := conn.Channel()
	failOnError(err, "Erro ao abrir channel de publish")
	defer publishCh.Close()

	consumeCh, err := conn.Channel()
	failOnError(err, "Erro ao abrir channel de consume")
	defer consumeCh.Close()

	// Própria chave privada, usada para assinar tudo que publicarmos.
	chavePrivada, err := seguranca.CarregarChavePrivada("ms-entrega/chaves/entrega_private.pem")
	failOnError(err, "Erro ao carregar chave privada do MS Entrega")

	// Chaves públicas de todos os outros microsserviços
	chavesPublicas, err := seguranca.CarregarTodasChavesPublicas("ms-entrega/chaves")
	failOnError(err, "Erro ao carregar chaves públicas")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		consumirEProcessar(consumeCh, publishCh, chavePrivada, chavesPublicas)
	}()

	log.Printf(" [*] MS Entrega rodando. Aguardando pagamento.aprovado. CTRL+C para sair")
	wg.Wait()
}
