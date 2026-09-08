package main

import (
	//"context"
	"crypto/ed25519"
	//"e-commerce-sd/seguranca"
	"log"
	//"math/rand"
	//"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeExchange = "Promoções"
const NomeServico = "promocoes"

// PromocaoGerada é o payload publicado a cada promoção sorteada.
type PromocaoGerada struct {
	Produto  string
	Desconto int // percentual, ex: 10, 20, 30
}

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

// publicar segue seu padrão — assina e publica na exchange Promoções.
func publicar(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey,
	routingKey string, body interface{}) {

}

// sortearCategoria escolhe aleatoriamente entre "A", "B", "C".
func sortearCategoria() string {

}

// sortearPromocao monta um PromocaoGerada aleatório (produto e desconto
// fictícios, já que não há banco envolvido aqui).
func sortearPromocao() PromocaoGerada {

}

// rodarLoopDePromocoes fica publicando promoções em intervalos
// (ex: time.Sleep entre iterações), montando a routing key
// "promocao.categoria.<X>" a partir da categoria sorteada.
// É a única "lógica de negócio" real deste microsserviço.
func rodarLoopDePromocoes(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey) {

}

func main() {

}
