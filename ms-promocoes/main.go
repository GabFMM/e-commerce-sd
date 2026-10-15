package main

import (
	"bytes"
	"crypto/ed25519"
	"e-commerce-sd/data"
	"e-commerce-sd/ms-promocoes/service"
	"e-commerce-sd/seguranca"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeFila = "promocoes"

// Lida uma única vez no main()
var resendAPIKey string

// Cliente HTTP reaproveitado em todos os envios de e-mail
var httpClient = &http.Client{Timeout: 5 * time.Second}

// Encerra o MS
func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

// Catálogo usado só para sortear promoções. O MS Promoções tem banco próprio
// apenas para os interesses dos clientes (tabela interesses).
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
	Produto   string `json:"produto"`
	Desconto  int    `json:"desconto"`
	Categoria string `json:"categoria"`
}

// Payload do evento interesse.promocao publicado pelo MS Principal
type EventoInteresse struct {
	Email      string   `json:"email"`
	Categorias []string `json:"categorias"` // pode ser vazio/nulo: cliente sem nenhuma categoria
	Acao       string   `json:"acao"`       // "CADASTRAR" ou "CANCELAR"
}

func enviarEmailPromocao(destinatario string, prom PromocaoGerada) {
	assunto := fmt.Sprintf("Oferta Especial! %d%% de desconto em %s", prom.Desconto, prom.Produto)
	corpoHTML := fmt.Sprintf(`
		<h2>Sua categoria de interesse %s está em promoção!</h2>
		<p>Aproveite <strong>%d%% de desconto</strong> no produto <strong>%s</strong>.</p>
		<p>Acesse nosso site e garanta o seu!</p>
	`, prom.Categoria, prom.Desconto, prom.Produto)

	payload := map[string]interface{}{
		"from":    "Promocoes <onboarding@resend.dev>",
		"to":      []string{destinatario},
		"subject": assunto,
		"html":    corpoHTML,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[ERRO-EMAIL] Falha ao montar payload para %s: %v", destinatario, err)
		return
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Printf("[ERRO-EMAIL] Falha ao criar requisição para %s: %v", destinatario, err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+resendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[ERRO-EMAIL] Falha ao disparar e-mail para %s: %v", destinatario, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("[EMAIL-ENVIADO] E-mail entregue com sucesso para %s!", destinatario)
		return
	}

	// Lê um pedaço da resposta para facilitar o diagnóstico (chave inválida, domínio não verificado etc.)
	detalhe, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	log.Printf("[ERRO-EMAIL] API Externa retornou Status %d para %s: %s", resp.StatusCode, destinatario, string(detalhe))
}

func sortearPromocao() PromocaoGerada {
	categorias := []string{"A", "B", "C"}
	categoria := categorias[rand.Intn(len(categorias))]

	var produtosDaCategoria []string
	for prod, cat := range Produtos {
		if cat == categoria {
			produtosDaCategoria = append(produtosDaCategoria, prod)
		}
	}

	produtoSorteado := produtosDaCategoria[rand.Intn(len(produtosDaCategoria))]
	descontoSorteado := (rand.Intn(9) * 5) + 10 // 10% a 50%

	return PromocaoGerada{
		Produto:   produtoSorteado,
		Desconto:  descontoSorteado,
		Categoria: categoria,
	}
}

func rodarLoopDePromocoes() {
	for {
		prom := sortearPromocao()
		log.Printf("[PROMOÇÃO-GERADA] Categoria %s | Produto: %s | Desconto: %d%%", prom.Categoria, prom.Produto, prom.Desconto)

		// Busca no banco os e-mails que têm a categoria sorteada na lista deles
		emailsInteressados, err := service.ObterEmailsPorCategoria(prom.Categoria)
		switch {
		case err != nil:
			log.Printf("[ERRO-PROMOCOES] Falha ao consultar interessados da categoria %s: %v", prom.Categoria, err)
		case len(emailsInteressados) > 0:
			log.Printf("[INFO-PROMOCOES] Encontrados %d clientes interessados na categoria %s", len(emailsInteressados), prom.Categoria)
			for _, email := range emailsInteressados {
				go enviarEmailPromocao(email, prom)
			}
		default:
			log.Printf("[INFO-PROMOCOES] Nenhum cliente cadastrado para a categoria %s no momento.", prom.Categoria)
		}

		tempoEspera := rand.Intn(11) + 20 // entre 20 e 30 segundos
		time.Sleep(time.Duration(tempoEspera) * time.Second)
	}
}

func consumirMensagensPromocao(
	msgs <-chan amqp.Delivery,
	chavesPublicas map[string]ed25519.PublicKey,
) {
	for d := range msgs {
		payload, err := seguranca.AbrirPacote(d.Body, chavesPublicas)
		if err != nil {
			log.Printf("[ERRO-PROMOCOES] Erro ao abrir pacote: %v", err)
			continue
		}

		var ev EventoInteresse
		if err := json.Unmarshal(payload, &ev); err != nil {
			log.Printf("[ERRO-PROMOCOES] Erro ao desserializar: %v", err)
			continue
		}

		switch ev.Acao {
		case "CADASTRAR":
			// Substitui a lista de categorias do e-mail (lista vazia = cadastrado sem categorias)
			if err := service.CadastrarInteresse(ev.Email, ev.Categorias); err != nil {
				log.Printf("[ERRO-PROMOCOES] Falha ao cadastrar interesse de %s: %v", ev.Email, err)
			}
		case "CANCELAR":
			if err := service.RemoverInteresse(ev.Email); err != nil {
				log.Printf("[ERRO-PROMOCOES] Falha ao cancelar interesse de %s: %v", ev.Email, err)
			}
		default:
			log.Printf("[ERRO-PROMOCOES] Ação desconhecida %q para %s", ev.Acao, ev.Email)
		}
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[AVISO-PROMOCOES] Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}
	resendAPIKey = os.Getenv("RESEND_API_KEY")
	if resendAPIKey == "" {
		log.Println("[AVISO-PROMOCOES] RESEND_API_KEY vazia: os envios de e-mail vão falhar")
	}

	// Conecta já na subida: se o banco estiver fora, falha agora e não na 1ª mensagem
	data.Pool("PROMOCOES")
	defer data.FecharBancos()

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Erro ao conectar ao RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Erro ao abrir channel")
	defer ch.Close()

	msgs, err := ch.Consume(NomeFila, "", true, false, false, false, nil)
	failOnError(err, "Erro ao registrar consumidor na fila promocoes")

	// Só precisa das chaves públicas: este MS apenas verifica assinaturas
	chavesPublicas, err := seguranca.CarregarTodasChavesPublicas("ms-promocoes/chaves/")
	failOnError(err, "Erro ao carregar chaves públicas")

	// Consome o evento interesse.promocao
	go consumirMensagensPromocao(msgs, chavesPublicas)

	// Gera promoções periodicamente e dispara os e-mails
	go rodarLoopDePromocoes()

	log.Printf("[INFO-MS-PROMOCOES] MS Promoções rodando e ouvindo RabbitMQ...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
