package main

import (
	"crypto/ed25519"
	"e-commerce-sd/seguranca"
	"encoding/json"
	"log"
	"os"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PromocaoGerada struct {
	Produto   string
	Desconto  int
	Categoria string
}

const NomeExchange = "Promoções"

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func printOnError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}

// tratarPromocao recebe o payload já validado e só imprime a promoção recebida.
func tratarPromocao(nomeFila, routingKey string, payload json.RawMessage) {
	var promocao PromocaoGerada
	if err := json.Unmarshal(payload, &promocao); err != nil {
		printOnError(err, "[ERRO-CONSUMER] Falha ao desserializar payload")
		return
	}

	log.Printf("[%s] Promoção recebida (%s): %s - %d%% de desconto", nomeFila, routingKey, promocao.Produto, promocao.Desconto)
}

// consumir registra o consumidor na fila Q1 e despacha cada
// mensagem recebida para tratarPromocao.
func consumir(consumeCh *amqp.Channel, nomeFila string, chavesPublicas map[string]ed25519.PublicKey) {
	msgs, err := consumeCh.Consume(
		nomeFila, // queue
		"",       // consumer
		true,     // auto-ack
		false,    // exclusive
		false,    // no-local
		false,    // no-wait
		nil,      // args
	)
	failOnError(err, "[ERRO-CONSUMER] Falha ao registrar consumidor na fila "+nomeFila)

	for d := range msgs {
		payload, err := seguranca.AbrirPacote(d.Body, chavesPublicas)
		if err != nil {
			log.Printf("[ERRO-CONSUMER] Evento descartado (%s): %v", d.RoutingKey, err)
			continue
		}
		tratarPromocao(nomeFila, d.RoutingKey, payload)
	}
}

func main() {

	// Valida se o argumento do nome da fila foi passado na execução
	if len(os.Args) < 2 {
		log.Panicf("[ERRO] Informe a fila esperada. Exemplo: go run main.go Q1")
	}

	nomeFila := os.Args[1]
	if nomeFila != "Q1" && nomeFila != "Q2" {
		log.Panicf("[ERRO] Fila inválida '%s'. Use apenas 'Q1' ou 'Q2'.", nomeFila)
	}

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Erro ao conectar ao RabbitMQ")
	defer conn.Close()

	consumeCh, err := conn.Channel()
	failOnError(err, "Erro ao abrir channel de consume")
	defer consumeCh.Close()

	chavesPublicas, err := seguranca.CarregarTodasChavesPublicas("consumidor/chaves")
	failOnError(err, "Erro ao carregar chaves públicas do consumidor")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		consumir(consumeCh, nomeFila, chavesPublicas)
	}()

	log.Printf(" [*] Consumidor rodando na fila %s. CTRL+C para sair", nomeFila)
	wg.Wait()

}
