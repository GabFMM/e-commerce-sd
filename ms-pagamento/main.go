package main

import (
	"context"
	"crypto/ed25519"
	"e-commerce-sd/seguranca"
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeExchange = "eCommerce"
const NomeServico = "pagamento"
const NomeFila = "pagamento"

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

const ProbabilidadeAprovacao = 7

// PedidoEstoqueOk é o que esperamos no payload do evento pedido.estoque_ok.
type PedidoEstoqueOk struct {
	PedidoID int `json:"pedido_id"`
}

// PagamentoResultado é o payload publicado tanto para aprovado
// quanto para recusado — a routing key diferencia o resultado.
type PagamentoResultado struct {
	PedidoID int `json:"pedido_id"`
}

func publicarResultado(ch *amqp.Channel, evento PagamentoResultado, routingKey string,
	chavePrivada ed25519.PrivateKey) {
	pacoteBytes, err := seguranca.CriarPacote(evento, NomeServico, chavePrivada)
	if err != nil {
		log.Printf(" [ERRO] falha ao criar pacote assinado: %v ", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = ch.PublishWithContext(ctx,
		NomeExchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        pacoteBytes,
		},
	)

	if err != nil {
		log.Printf(" [ERRO] Falha ao publicar %s: %v", routingKey, err)
		return
	}

	log.Printf(" [LOG] Publicado %s para pedido %s", routingKey, evento.PedidoID)
}

func handlePedidoEstoqueOk(payload []byte, publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey) {
	var dados PedidoEstoqueOk
	if err := json.Unmarshal(payload, &dados); err != nil {
		log.Printf(" [ERRO] Falha ao desserializar payload: %v", err)
		return
	}

	log.Printf(" [x] Processando pagamento do pedido %s...", dados.PedidoID)
	time.Sleep(300 * time.Millisecond) // simula tempo de processamento

	aprovado := rand.Int()%10+1 < ProbabilidadeAprovacao

	evento := PagamentoResultado{PedidoID: dados.PedidoID}

	routingKey := "pagamento.recusado"
	if aprovado {
		routingKey = "pagamento.aprovado"
	}

	publicarResultado(publishCh, evento, routingKey, chavePrivada)
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

		switch d.RoutingKey {
		case "pedido.estoque_ok":
			handlePedidoEstoqueOk(payload, publishCh, chavePrivada)
		default:
			log.Printf(" [!] Routing key inesperada na fila %s: %s", NomeFila, d.RoutingKey)
		}
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Erro ao conectar ao RabbitMQ")
	defer conn.Close()

	publishCh, err := conn.Channel()
	failOnError(err, "Erro ao abrir channel de publish")
	defer publishCh.Close()

	consumeCh, err := conn.Channel()
	failOnError(err, "Erro ao abrir channel de consume")
	defer consumeCh.Close()

	chavePrivada, err := seguranca.CarregarChavePrivada("ms-pagamento/chaves/pagamento_private.pem")
	failOnError(err, "Erro ao carregar chave privada do MS Pagamento")

	chavesPublicas, err := seguranca.CarregarTodasChavesPublicas("ms-pagamento/chaves")
	failOnError(err, "Erro ao carregar chaves públicas")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		consumirEProcessar(consumeCh, publishCh, chavePrivada, chavesPublicas)
	}()

	log.Printf(" [*] MS Pagamento rodando. Aguardando pedido.estoque_ok. CTRL+C para sair")
	wg.Wait()
}
