package main

import (
	"context"
	"crypto/ed25519"
	"e-commerce-sd/seguranca"
	"log"
	"math/rand"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeExchange = "Promoções"
const NomeServico = "promocoes"

var Produtos = map[string]string{
	"Notebook":   "A",
	"Mouse":      "B",
	"Teclado":    "B",
	"Monitor":    "A",
	"Webcam":     "C",
	"Headset":    "C",
	"Impressora": "A",
	"Cabo HDMI":  "B",
}

// PromocaoGerada é o payload publicado a cada promoção sorteada.
type PromocaoGerada struct {
	Produto   string
	Desconto  int // percentual, ex: 10, 20, 30
	Categoria string
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
	routingKey string, body PromocaoGerada) {
	pacotesBytes, err := seguranca.CriarPacote(body, NomeServico, chavePrivada)
	if err != nil {
		log.Printf(" [ERRO] falha ao criar pacote assinado: %v no serviço %s", err, NomeServico)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = publishCh.PublishWithContext(
		ctx,
		NomeExchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        pacotesBytes,
		},
	)

	if err != nil {
		log.Printf(" [ERRO] Falha ao publicar %s: %v", routingKey, err)
		return
	}

	log.Printf(" [LOG] Publicado %s para pedido %s", routingKey, body.Produto)
}

// sortearCategoria escolhe aleatoriamente entre "A", "B", "C".
func sortearCategoria() string {
	var sorteado int = rand.Int() % 3

	switch sorteado {
	case 0:
		return "A"
	case 1:
		return "B"
	case 2:
		return "C"
	default:
		return "A"
	}
}

// sortearPromocao monta um PromocaoGerada aleatório (produto e desconto
// fictícios, já que não há banco envolvido aqui).
func sortearPromocao() PromocaoGerada {
	categoria := sortearCategoria()

	//Filtrar os produtos que pertencem à categoria sorteada
	var produtosDaCategoria []string
	for produto, categ := range Produtos {
		if categ == categoria {
			produtosDaCategoria = append(produtosDaCategoria, produto)
		}
	}

	// Sortear um produto da lista filtrada
	indiceProduto := rand.Intn(len(produtosDaCategoria))
	produtoSorteado := produtosDaCategoria[indiceProduto]

	// Sortear um valor de desconto ( entre 10% e 50%, de 5 em 5)
	// rand.Intn(9) gera de 0 a 8. (0*5)+10 = 10% até (8*5)+10 = 50%
	descontoSorteado := (rand.Intn(9) * 5) + 10

	// Montar e retornar a struct
	return PromocaoGerada{
		Produto:   produtoSorteado,
		Desconto:  descontoSorteado,
		Categoria: categoria,
	}

}

// rodarLoopDePromocoes fica publicando promoções em intervalos
// (time.Sleep entre iterações), montando a routing key
// "promocao.categoria.<X>" a partir da categoria sorteada.
func rodarLoopDePromocoes(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey) {
	for {
		promGer := sortearPromocao()
		routing := "promocao.categoria." + promGer.Categoria
		publicar(publishCh, chavePrivada, routing, promGer)

		// Rand.Intn(16) gera de 0 a 15 + 5 = intervalo exato entre 5 e 20 segundos
		tempoEspera := rand.Intn(16) + 5
		log.Printf(" [*] Próxima promoção em %d segundos...", tempoEspera)
		time.Sleep(time.Second * time.Duration(tempoEspera))
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Erro ao conectar ao RabbitMQ")
	defer conn.Close()

	publishCh, err := conn.Channel()
	failOnError(err, "Erro ao abrir channel de publish")
	defer publishCh.Close()

	// Ajustado para apontar para a chave correspondente ao serviço de promoções
	chavePrivada, err := seguranca.CarregarChavePrivada("chaves/promocoes_private.pem")
	failOnError(err, "Erro ao carregar chave privada do MS Promocoes")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		rodarLoopDePromocoes(publishCh, chavePrivada)
	}()

	log.Printf(" [*] MS Promocoes rodando. Pressione CTRL+C para sair.")
	wg.Wait()
}
