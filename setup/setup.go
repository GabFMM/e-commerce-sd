// Arquivo para configurar as filas, routing keys e a exchange, dai cada microsserviço vai ter que
// enviar e consumir, enviar direto para a exchange e consumir de alguma fila especifica

package main

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Struct igual de C para a gente relacionar Fila-> routingkeys
type QueueRoutingKeys struct {
	NomeQueue   string
	RoutingKeys []string
}

var Filas = []QueueRoutingKeys{
	{NomeQueue: "estoque", RoutingKeys: []string{"pedido_criado", "pedido_excluido"}},
	{NomeQueue: "principal", RoutingKeys: []string{"pedido_estoque_ok", "estoque_indisponivel", "pagamento_aprovado", "pagamento_reprovado", "pedido_enviado"}},
	{NomeQueue: "pagamento", RoutingKeys: []string{"pedido_estoque_ok"}},
	{NomeQueue: "entrega", RoutingKeys: []string{"pagamento_aprovado"}},
}

const NomeExchange = "eCommerce"

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {

	//Aqui foi só um teste ver se tava certo a struct
	// for _, routing := range Filas {
	// 	log.Printf("%s: %s", routing.NomeQueue, routing.RoutingKeys)
	// }

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Erro ao criar conexão com servidor local do amqp")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Erro ao criar channel com o servidor")
	defer ch.Close()

	//Declarar a exchange e manter ela duravel
	err = ch.ExchangeDeclare(
		NomeExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)

	failOnError(err, "Falha ao declarar a exchange")
	log.Printf("[LOG] Exchange %q declarada", NomeExchange)

	//Loop para criar as filas
	for _, fila := range Filas {
		queue, err := ch.QueueDeclare(
			fila.NomeQueue,
			true,
			false,
			false,
			false,
			nil,
		)
		failOnError(err, "Falha ao declarar a fila: "+fila.NomeQueue)
		log.Printf(" [x] Queue %q declarada", queue.Name)

		//Dentro do loop de filas um loop para bindar as routing keys
		for _, routKey := range fila.RoutingKeys {
			err = ch.QueueBind(
				queue.Name,
				routKey,
				NomeExchange,
				false,
				nil,
			)
			failOnError(err, "Falha ao bindar "+queue.Name+" com a routing key "+routKey)
			log.Printf("     -> bind com routing key %q", routKey)
		}
	}
	log.Println(" [x] Setup concluído com sucesso.")
}

// Listar exchanges
// sudo rabbitmqctl list_exchanges name type

// Listar filas (nome e quantidade de mensagens)
// sudo rabbitmqctl list_queues name messages

// Litar bindings
// sudo rabbitmqctl list_bindings
//Eu vi em uma interface visual que o rabbit tem tambem legal
