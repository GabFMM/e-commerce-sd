package main

import (
	//"crypto/ed25519"
	//"e-commerce-sd/seguranca"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeExchange = "Promoções"
const NomeFila = "Q2"

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

// tratarPromocao recebe o body cru (ou já verificado, se decidirem
// validar assinatura) e só imprime/loga a promoção recebida.
// C1/C2 não publicam nada, então não precisam de publishCh nem
// da função publicar.
func tratarPromocao(routingKey string, body []byte)

// consumir registra o consumidor na fila Q1 e despacha cada
// mensagem recebida para tratarPromocao.
func consumir(consumeCh *amqp.Channel)

func main()
