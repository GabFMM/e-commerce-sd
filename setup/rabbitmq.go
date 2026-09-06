// Arquivo para configurar as filas, routing keys e a exchange, dai cada microsserviço vai ter que
// enviar e consumir, enviar direto para a exchange e consumir de alguma fila especifica

package setup

import(
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Struct igual de C para a gente relacionar Fila-> routingkeys
type QueueRoutingKeys struct {
	NomeQueue   string
	RoutingKeys []string
}

var Filas = []QueueRoutingKeys{
	{NomeQueue: "estoque", RoutingKeys: []string{"pedido.criado", "pedido.excluido"}},
	{NomeQueue: "principal", RoutingKeys: []string{"pedido.estoque_ok", "estoque.indisponivel", "pagamento.aprovado", "pagamento.recusado", "pedido.enviado"}},
	{NomeQueue: "pagamento", RoutingKeys: []string{"pedido.estoque_ok"}},
	{NomeQueue: "entrega", RoutingKeys: []string{"pagamento.aprovado"}},
}

const NomeExchange = "eCommerce"

func ConfigurarFilas() {
	//Aqui foi só um teste ver se tava certo a struct
	// for _, routing := range Filas {
	// 	log.Printf("%s: %s", routing.NomeQueue, routing.RoutingKeys)
	// }

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Panicf("%s: %s", "[ERRO] Erro ao criar conexão com servidor local do amqp", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Panicf("%s: %s", "[ERRO] Erro ao criar channel com o servidor", err)
	}
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

	if err != nil {
		log.Panicf("%s: %s", "[ERRO] Falha ao declarar a exchange", err)
	}
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
		if err != nil {
			log.Panicf("%s: %s", "[ERRO] Falha ao declarar a fila: "+fila.NomeQueue, err)
		}
		log.Printf("[LOG] Queue %q declarada", queue.Name)

		//Dentro do loop de filas um loop para bindar as routing keys e o exchange
		for _, routKey := range fila.RoutingKeys {
			err = ch.QueueBind(
				queue.Name,
				routKey,
				NomeExchange,
				false,
				nil,
			)
			if err != nil {
				log.Panicf("%s: %s", "[ERRO] Falha ao bindar "+queue.Name+" com a routing key "+routKey, err)
			}
			log.Printf("     -> bind com routing key %q", routKey)
		}
	}
}