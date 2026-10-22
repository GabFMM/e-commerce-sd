package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"e-commerce-sd/ms-promocoes/service"
	"e-commerce-sd/seguranca"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/resend/resend-go/v2"
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

type PromocaoGerada struct {
	Produto   string `json:"Produto"`
	Desconto  int    `json:"Desconto"`
	Categoria string `json:"Categoria"`
}

type RequisicaoInteresse struct {
	Email      string   `json:"email"`
	Categorias []string `json:"categorias"`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func publicar(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, routingKey string, body PromocaoGerada) {
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

func sortearPromocao() PromocaoGerada {
	categoria := sortearCategoria()

	var produtosDaCategoria []string
	for produto, categ := range Produtos {
		if categ == categoria {
			produtosDaCategoria = append(produtosDaCategoria, produto)
		}
	}

	indiceProduto := rand.Intn(len(produtosDaCategoria))
	produtoSorteado := produtosDaCategoria[indiceProduto]
	descontoSorteado := (rand.Intn(9) * 5) + 10

	return PromocaoGerada{
		Produto:   produtoSorteado,
		Desconto:  descontoSorteado,
		Categoria: categoria,
	}
}

// rodarLoopDePromocoes sorteia, publica, busca os e-mails e dispara via Resend diretamente.
func rodarLoopDePromocoes(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, resendApiKey string) {
	client := resend.NewClient(resendApiKey)

	for {
		promGer := sortearPromocao()
		routing := "promocao.categoria." + promGer.Categoria

		// 1. Publica na Exchange "Promoções" no RabbitMQ
		publicar(publishCh, chavePrivada, routing, promGer)

		// 2. Busca diretamente no banco os e-mails cadastrados nessa categoria
		emails, err := service.ObterEmailsPorCategoria(promGer.Categoria)
		if err != nil {
			log.Printf(" [ERRO BD] Falha ao consultar e-mails da categoria %s: %v", promGer.Categoria, err)
		} else if len(emails) == 0 {
			log.Printf(" [INFO] Nenhum usuário cadastrado para a categoria %s", promGer.Categoria)
		} else {
			// 3. Dispara os e-mails via Resend
			log.Printf(" [RESEND] Disparando e-mails para %d usuário(s) da categoria %s...", len(emails), promGer.Categoria)
			for _, email := range emails {
				params := &resend.SendEmailRequest{
					From:    "Promocoes <onboarding@resend.dev>",
					To:      []string{email},
					Subject: fmt.Sprintf("Promoção Imperdível: %s com %d%% de Desconto!", promGer.Produto, promGer.Desconto),
					Html:    fmt.Sprintf("<h1>Nova Promoção!</h1><p>O item <strong>%s</strong> da categoria <strong>%s</strong> está com desconto especial de <strong>%d%%</strong>!</p>", promGer.Produto, promGer.Categoria, promGer.Desconto),
				}

				sent, err := client.Emails.Send(params)
				if err != nil {
					log.Printf(" [ERRO RESEND] Falha ao enviar e-mail para %s: %v", email, err)
					continue
				}
				log.Printf(" [RESEND] E-mail enviado com sucesso para %s | ID: %s", email, sent.Id)
			}
		}

		tempoEspera := rand.Intn(16) + 5
		log.Printf(" [*] Próxima promoção em %d segundos...", tempoEspera)
		time.Sleep(time.Second * time.Duration(tempoEspera))
	}
}

func extrairInteresse(body []byte) (RequisicaoInteresse, error) {
	var req RequisicaoInteresse

	var envelope struct {
		Conteudo json.RawMessage `json:"conteudo"`
		Payload  json.RawMessage `json:"payload"`
		Dados    json.RawMessage `json:"dados"`
		Data     json.RawMessage `json:"data"`
		Body     json.RawMessage `json:"body"`
	}

	if err := json.Unmarshal(body, &envelope); err == nil {
		campos := []json.RawMessage{envelope.Conteudo, envelope.Payload, envelope.Dados, envelope.Data, envelope.Body}
		for _, raw := range campos {
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &req); err == nil && req.Email != "" {
					return req, nil
				}
				var strPayload string
				if err := json.Unmarshal(raw, &strPayload); err == nil {
					if err := json.Unmarshal([]byte(strPayload), &req); err == nil && req.Email != "" {
						return req, nil
					}
				}
			}
		}
	}

	if err := json.Unmarshal(body, &req); err == nil && req.Email != "" {
		return req, nil
	}

	return req, fmt.Errorf("impossível extrair interesse de: %s", string(body))
}

// Consumidor para registrar os e-mails e categorias no PostgreSQL vindos do MS Principal
func iniciarConsumidorDeInteresses(ch *amqp.Channel) {
	const exchangePrincipal = "eCommerce"

	err := ch.ExchangeDeclare(
		exchangePrincipal,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Erro ao declarar exchange eCommerce")

	q, err := ch.QueueDeclare(
		"promocoes",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Erro ao declarar fila de cadastro de interesses")

	err = ch.QueueBind(
		q.Name,
		"interesse.promocao",
		exchangePrincipal,
		false,
		nil,
	)
	failOnError(err, "Erro ao vincular fila de interesses à exchange eCommerce")

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Erro ao iniciar consumidor de interesses")

	go func() {
		for d := range msgs {
			req, err := extrairInteresse(d.Body)
			if err != nil {
				log.Printf(" [ERRO CONSUMIDOR INTERESSE] %v", err)
				continue
			}

			err = service.CadastrarInteresse(req.Email, req.Categorias)
			if err != nil {
				log.Printf(" [ERRO BD] Falha ao registrar interesse: %v", err)
			} else {
				log.Printf(" [INTERESSE REGISTRADO] Email: %s | Categorias: %v", req.Email, req.Categorias)
			}
		}
	}()
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Erro ao conectar ao RabbitMQ")
	defer conn.Close()

	publishCh, err := conn.Channel()
	failOnError(err, "Erro ao abrir channel de publicação")
	defer publishCh.Close()

	consumeCh, err := conn.Channel()
	failOnError(err, "Erro ao abrir channel de consumo")
	defer consumeCh.Close()

	chavePrivada, err := seguranca.CarregarChavePrivada("ms-promocoes/chaves/promocoes_private.pem")
	failOnError(err, "Erro ao carregar chave privada do MS Promocoes")

	resendApiKey := os.Getenv("RESEND_API_KEY")
	if resendApiKey == "" {
		resendApiKey = "re_SUA_CHAVE_AQUI"
	}

	// Mantém ativo apenas o consumidor de novos cadastros de interesse do frontend
	iniciarConsumidorDeInteresses(consumeCh)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		rodarLoopDePromocoes(publishCh, chavePrivada, resendApiKey)
	}()

	log.Printf(" [*] MS Promocoes rodando. Pressione CTRL+C para sair.")
	wg.Wait()
}
