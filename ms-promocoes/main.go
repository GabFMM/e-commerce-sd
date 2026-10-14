package main

import (
	"bytes"
	"crypto/ed25519"
	"e-commerce-sd/seguranca"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
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

var categoriasValidas = map[string]bool{
	"A": true,
	"B": true,
	"C": true,
}

type PromocaoGerada struct {
	Produto   string `json:"produto"`
	Desconto  int    `json:"desconto"`
	Categoria string `json:"categoria"`
}

// Payload do evento interesse.promocao publicado pelo MS Principal
type EventoInteresse struct {
	Email      string   `json:"email"`
	Categorias []string `json:"categorias"`
	Acao       string   `json:"acao"` // "CADASTRAR" ou "CANCELAR"
}

type GerenciadorInteresse struct {
	mu       sync.RWMutex
	clientes map[string]map[string]bool // email -> set de categorias
}

func NovoGerenciadorInteresse() *GerenciadorInteresse {
	return &GerenciadorInteresse{
		clientes: make(map[string]map[string]bool),
	}
}

func normalizarEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (g *GerenciadorInteresse) Cadastrar(email string, categorias []string) {
	emailNorm := normalizarEmail(email)

	var validas []string
	for _, cat := range categorias {
		catUpper := strings.ToUpper(strings.TrimSpace(cat))
		if categoriasValidas[catUpper] {
			validas = append(validas, catUpper)
		}
	}

	if len(validas) == 0 {
		log.Printf("[AVISO-PROMOCOES] Cadastro ignorado para %s: nenhuma categoria válida em %v", emailNorm, categorias)
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	set, existe := g.clientes[emailNorm]
	if !existe {
		set = make(map[string]bool)
		g.clientes[emailNorm] = set
	}

	for _, cat := range validas {
		set[cat] = true
	}

	log.Printf("[INFO-PROMOCOES] Interesse registrado. Email: %s | Categorias: %v | Novo cliente: %t",
		emailNorm, validas, !existe)
}

func (g *GerenciadorInteresse) Remover(email string) {
	emailNorm := normalizarEmail(email)

	g.mu.Lock()
	defer g.mu.Unlock()

	if _, existe := g.clientes[emailNorm]; existe {
		delete(g.clientes, emailNorm)
		log.Printf("[INFO-PROMOCOES] Interesse removido. Email: %s", emailNorm)
		return
	}
	log.Printf("[INFO-PROMOCOES] Cancelamento ignorado: %s não estava cadastrado", emailNorm)
}

func (g *GerenciadorInteresse) ObterEmailsPorCategoria(categoria string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	catUpper := strings.ToUpper(strings.TrimSpace(categoria))
	var resultado []string

	for email, setCategorias := range g.clientes {
		if setCategorias[catUpper] {
			resultado = append(resultado, email)
		}
	}

	return resultado
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

func rodarLoopDePromocoes(gerenciador *GerenciadorInteresse) {
	for {
		prom := sortearPromocao()
		log.Printf("[PROMOÇÃO-GERADA] Categoria %s | Produto: %s | Desconto: %d%%", prom.Categoria, prom.Produto, prom.Desconto)

		// Notifica os e-mails cadastrados na categoria sorteada
		emailsInteressados := gerenciador.ObterEmailsPorCategoria(prom.Categoria)
		if len(emailsInteressados) > 0 {
			log.Printf("[INFO-PROMOCOES] Encontrados %d clientes interessados na categoria %s", len(emailsInteressados), prom.Categoria)
			for _, email := range emailsInteressados {
				go enviarEmailPromocao(email, prom)
			}
		} else {
			log.Printf("[INFO-PROMOCOES] Nenhum cliente cadastrado para a categoria %s no momento.", prom.Categoria)
		}

		tempoEspera := rand.Intn(11) + 20 // entre 20 e 30 segundos
		time.Sleep(time.Duration(tempoEspera) * time.Second)
	}
}

func consumirMensagensPromocao(
	msgs <-chan amqp.Delivery,
	chavesPublicas map[string]ed25519.PublicKey,
	gerenciador *GerenciadorInteresse,
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
			gerenciador.Cadastrar(ev.Email, ev.Categorias)
		case "CANCELAR":
			gerenciador.Remover(ev.Email)
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

	gerenciador := NovoGerenciadorInteresse()

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
	go consumirMensagensPromocao(msgs, chavesPublicas, gerenciador)

	// Gera promoções periodicamente e dispara os e-mails
	go rodarLoopDePromocoes(gerenciador)

	log.Printf("[INFO-MS-PROMOCOES] MS Promoções rodando e ouvindo RabbitMQ...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
