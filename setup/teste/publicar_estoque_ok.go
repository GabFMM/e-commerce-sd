package main

import (
	"context"
	"e-commerce-sd/seguranca"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PedidoEstoqueOk struct {
	PedidoID string `json:"pedido_id"`
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	chavePrivadaEstoque, err := seguranca.CarregarChavePrivada("../../ms-estoque/chaves/estoque_private.pem")
	if err != nil {
		log.Fatal(err)
	}

	eventos := []PedidoEstoqueOk{
		{PedidoID: "123"},
		{PedidoID: "456"},
		{PedidoID: "789"},
		{PedidoID: "101"},
		{PedidoID: "201"},
	}

	for _, evento := range eventos {
		// Monta o pacote assinado PARA CADA evento individualmente
		pacote, err := seguranca.CriarPacote(evento, "estoque", chavePrivadaEstoque)
		if err != nil {
			log.Printf("falha ao criar pacote para pedido %s: %v", evento.PedidoID, err)
			continue // pula este, tenta o próximo
		}

		// Publish fica FORA do if — roda sempre que CriarPacote deu certo
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = ch.PublishWithContext(ctx,
			"eCommerce",
			"pedido.estoque_ok",
			false,
			false,
			amqp.Publishing{ContentType: "application/json", Body: pacote},
		)
		cancel()

		if err != nil {
			log.Printf("falha ao publicar pedido %s: %v", evento.PedidoID, err)
			continue
		}

		log.Printf(" [x] Evento pedido.estoque_ok publicado para pedido %s", evento.PedidoID)
		time.Sleep(1 * time.Second) // espaça as publicações pra ver uma por uma
	}
}
