package main

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeExchange = "eCommerce"
const NomeServico = "estoque"
const NomeFila = "estoque"

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
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
}
