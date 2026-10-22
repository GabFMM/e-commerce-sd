package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"e-commerce-sd/seguranca"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeExchange = "eCommerce"
const NomeServico = "pagamento"
const NomeFila = "pagamento"

// Encerra o MS
func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

// Lê uma variável de ambiente, ou devolve o valor padrão.
func env(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}

// URL do Mock de Pagamento e a URL deste MS que o Mock chamará (webhook).
var (
	mockURL    = env("MOCK_PAGAMENTO_URL", "http://localhost:8082")
	webhookURL = env("PAGAMENTO_WEBHOOK_URL", "http://localhost:8083/webhook/pagamento")
	httpAddr   = env("PAGAMENTO_ADDR", ":8083")
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

// Payload de entrada (pedido.estoque_ok) e de saída (pagamento.aprovado/recusado).
type PedidoDTO struct {
	PedidoID int `json:"pedido_id"`
}

// Payload de pagamento.criado: leva ao frontend (via Gateway/SSE) a URL do checkout.
type PagamentoCriado struct {
	PedidoID    int    `json:"pedido_id"`
	UrlCheckout string `json:"url_checkout"`
}

// Contrato com o Mock: POST {mock}/cobrancas
type cobrancaRequest struct {
	PedidoID   int    `json:"pedido_id"`
	WebhookURL string `json:"webhook_url"`
}

type cobrancaResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

// Contrato com o Mock: POST {webhook} (chamado pelo Mock quando o usuário clica)
type webhookRequest struct {
	PedidoID int    `json:"pedido_id" binding:"required"`
	Status   string `json:"status" binding:"required"` // "APROVADO" ou "RECUSADO"
}

type servico struct {
	publishCh    *amqp.Channel
	chavePrivada ed25519.PrivateKey

	// O *amqp.Channel não é thread-safe para publicar
	publishMu sync.Mutex

	// Pedidos com cobrança criada e ainda sem resposta do webhook.
	pendentesMu sync.Mutex
	pendentes   map[int]bool
}

func (s *servico) marcarPendente(id int) {
	s.pendentesMu.Lock()
	defer s.pendentesMu.Unlock()
	s.pendentes[id] = true
}

// Remove da lista de pendentes. Retorna false se o pedido não estava pendente.
func (s *servico) consumirPendente(id int) bool {
	s.pendentesMu.Lock()
	defer s.pendentesMu.Unlock()
	if !s.pendentes[id] {
		return false
	}
	delete(s.pendentes, id)
	return true
}

func (s *servico) publicar(routingKey string, body interface{}) error {
	pacote, err := seguranca.CriarPacote(body, NomeServico, s.chavePrivada)
	if err != nil {
		return fmt.Errorf("criar pacote: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.publishMu.Lock()
	defer s.publishMu.Unlock()

	err = s.publishCh.PublishWithContext(ctx, NomeExchange, routingKey, false, false,
		amqp.Publishing{ContentType: "application/json", Body: pacote})
	if err != nil {
		return fmt.Errorf("publicar %s: %w", routingKey, err)
	}

	log.Printf("[INFO-MS-PAGAMENTO] Publicado %s", routingKey)
	return nil
}

// Pede ao Mock a criação da cobrança e devolve a URL de checkout.
func criarCobranca(pedidoID int) (string, error) {
	corpo, err := json.Marshal(cobrancaRequest{PedidoID: pedidoID, WebhookURL: webhookURL})
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Post(mockURL+"/cobrancas", "application/json", bytes.NewReader(corpo))
	if err != nil {
		return "", fmt.Errorf("mock indisponível: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("mock respondeu status %d", resp.StatusCode)
	}

	var cobranca cobrancaResponse
	if err := json.NewDecoder(resp.Body).Decode(&cobranca); err != nil {
		return "", fmt.Errorf("resposta do mock inválida: %w", err)
	}
	if cobranca.CheckoutURL == "" {
		return "", errors.New("mock não devolveu checkout_url")
	}

	return cobranca.CheckoutURL, nil
}

// ---------- Consumo do RabbitMQ ----------

func (s *servico) handlePedidoEstoqueOk(payload json.RawMessage) {
	var pedido PedidoDTO
	if err := json.Unmarshal(payload, &pedido); err != nil {
		log.Printf("[ERRO-MS-PAGAMENTO] Falha ao desserializar payload: %v", err)
		return
	}

	log.Printf("[INFO-MS-PAGAMENTO] Criando cobrança do pedido %d...", pedido.PedidoID)

	urlCheckout, err := criarCobranca(pedido.PedidoID)
	if err != nil {
		// Sem cobrança o pedido ficaria parado para sempre (e com estoque reservado).
		// Tratamos como pagamento recusado: o Gateway exclui o pedido e o estoque libera a reserva.
		log.Printf("[ERRO-MS-PAGAMENTO] Falha ao criar cobrança do pedido %d: %v", pedido.PedidoID, err)
		if err := s.publicar("pagamento.recusado", pedido); err != nil {
			log.Printf("[ERRO-MS-PAGAMENTO] %v", err)
		}
		return
	}

	s.marcarPendente(pedido.PedidoID)

	evento := PagamentoCriado{PedidoID: pedido.PedidoID, UrlCheckout: urlCheckout}
	if err := s.publicar("pagamento.criado", evento); err != nil {
		log.Printf("[ERRO-MS-PAGAMENTO] %v", err)
	}
}

func (s *servico) consumir(consumeCh *amqp.Channel, chavesPublicas map[string]ed25519.PublicKey) {
	msgs, err := consumeCh.Consume(NomeFila, "", true, false, false, false, nil)
	failOnError(err, "[ERRO-MS-PAGAMENTO] Erro ao registrar consumidor na fila "+NomeFila)

	for d := range msgs {
		payload, err := seguranca.AbrirPacote(d.Body, chavesPublicas)
		if err != nil {
			log.Printf("[ERRO-MS-PAGAMENTO] Evento descartado (%s): %v", d.RoutingKey, err)
			continue
		}

		switch d.RoutingKey {
		case "pedido.estoque_ok":
			s.handlePedidoEstoqueOk(payload)
		default:
			log.Printf("[ERRO-MS-PAGAMENTO] Routing key inesperada: %s", d.RoutingKey)
		}
	}
}

// POST /webhook/pagamento  {"pedido_id": 12, "status": "APROVADO"}
func (s *servico) handleWebhook(c *gin.Context) {
	var req webhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	var routingKey string
	switch req.Status {
	case "APROVADO":
		routingKey = "pagamento.aprovado"
	case "RECUSADO":
		routingKey = "pagamento.recusado"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"erro": "status deve ser APROVADO ou RECUSADO"})
		return
	}

	if !s.consumirPendente(req.PedidoID) {
		c.JSON(http.StatusNotFound, gin.H{"erro": "cobrança não encontrada ou já processada"})
		return
	}

	log.Printf("[INFO-MS-PAGAMENTO] Webhook recebido: pedido %d %s", req.PedidoID, req.Status)

	if err := s.publicar(routingKey, PedidoDTO{PedidoID: req.PedidoID}); err != nil {
		log.Printf("[ERRO-MS-PAGAMENTO] %v", err)
		s.marcarPendente(req.PedidoID) // permite que o Mock tente de novo
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "falha ao publicar resultado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensagem": "ok"})
}

// ---------- main ----------

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "[ERRO-MS-PAGAMENTO] Erro ao conectar ao RabbitMQ")
	defer conn.Close()

	publishCh, err := conn.Channel()
	failOnError(err, "[ERRO-MS-PAGAMENTO] Erro ao abrir channel de publish")
	defer publishCh.Close()

	consumeCh, err := conn.Channel()
	failOnError(err, "[ERRO-MS-PAGAMENTO] Erro ao abrir channel de consume")
	defer consumeCh.Close()

	chavePrivada, err := seguranca.CarregarChavePrivada("ms-pagamento/chaves/pagamento_private.pem")
	failOnError(err, "[ERRO-MS-PAGAMENTO] Erro ao carregar chave privada")

	chavesPublicas, err := seguranca.CarregarTodasChavesPublicas("ms-pagamento/chaves")
	failOnError(err, "[ERRO-MS-PAGAMENTO] Erro ao carregar chaves públicas")

	s := &servico{
		publishCh:    publishCh,
		chavePrivada: chavePrivada,
		pendentes:    make(map[int]bool),
	}

	// Consumer RabbitMQ
	go s.consumir(consumeCh, chavesPublicas)

	// API HTTP (webhook)
	router := gin.Default()
	router.POST("/webhook/pagamento", s.handleWebhook)

	srv := &http.Server{Addr: httpAddr, Handler: router}
	go func() {
		log.Printf("[INFO-MS-PAGAMENTO] Webhook escutando em %s", httpAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Panicf("[ERRO-MS-PAGAMENTO] Erro no servidor HTTP: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[INFO-MS-PAGAMENTO] Encerrando...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
