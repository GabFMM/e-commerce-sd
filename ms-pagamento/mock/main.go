// Mock de pagamento (simula o provedor externo) para testes locais.
//
//	POST /cobrancas      {"pedido_id":1,"webhook_url":"..."} -> {"checkout_url":"http://localhost:4200/mock-pagamento/1"}
//	POST /checkout/:id   {"status":"APROVADO"|"RECUSADO"}    -> repassa o resultado ao webhook do MS Pagamento
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	addr = env("MOCK_PAGAMENTO_ADDR", ":8082")

	// Página do frontend (Angular) onde o usuário aprova ou recusa
	checkoutBaseURL = env("MOCK_CHECKOUT_BASE_URL", "http://localhost:4200/mock-pagamento")

	// Origem do frontend, liberada no CORS
	frontendOrigin = env("FRONTEND_ORIGIN", "http://localhost:4200")

	mu        sync.Mutex
	cobrancas = map[int]string{} // pedido_id -> webhook_url

	client = &http.Client{Timeout: 5 * time.Second}
)

func env(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}

type cobrancaRequest struct {
	PedidoID   int    `json:"pedido_id" binding:"required"`
	WebhookURL string `json:"webhook_url" binding:"required"`
}

// Chamado pelo MS Pagamento
func criarCobranca(c *gin.Context) {
	var req cobrancaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "pedido_id e webhook_url são obrigatórios"})
		return
	}

	mu.Lock()
	cobrancas[req.PedidoID] = req.WebhookURL
	mu.Unlock()

	log.Printf("[MOCK] Cobrança criada para o pedido %d", req.PedidoID)

	c.JSON(http.StatusCreated, gin.H{
		"checkout_url": fmt.Sprintf("%s/%d", checkoutBaseURL, 77),
	})
}

type resultadoRequest struct {
	Status string `json:"status" binding:"required,oneof=APROVADO RECUSADO"`
}

// Chamado pela página do Angular
func registrarResultado(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	var req resultadoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "status deve ser APROVADO ou RECUSADO"})
		return
	}

	mu.Lock()
	webhook, existe := cobrancas[id]
	mu.Unlock()
	if !existe {
		c.JSON(http.StatusNotFound, gin.H{"erro": "cobrança não encontrada ou já respondida"})
		return
	}

	corpo, _ := json.Marshal(gin.H{"pedido_id": id, "status": req.Status})
	resp, err := client.Post(webhook, "application/json", bytes.NewReader(corpo))
	if err != nil {
		log.Printf("[MOCK] Falha ao chamar o webhook do pedido %d: %v", id, err)
		c.JSON(http.StatusBadGateway, gin.H{"erro": "não foi possível chamar o webhook do MS Pagamento"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[MOCK] Webhook do pedido %d respondeu %d", id, resp.StatusCode)

		// O MS Pagamento não reconhece mais essa cobrança (já processada ou reiniciado)
		if resp.StatusCode == http.StatusNotFound {
			mu.Lock()
			delete(cobrancas, id)
			mu.Unlock()
		}

		c.JSON(http.StatusBadGateway, gin.H{
			"erro": fmt.Sprintf("o MS Pagamento respondeu status %d", resp.StatusCode),
		})
		return
	}

	mu.Lock()
	delete(cobrancas, id)
	mu.Unlock()

	log.Printf("[MOCK] Pedido %d: %s enviado ao webhook", id, req.Status)
	c.JSON(http.StatusOK, gin.H{"mensagem": "ok"})
}

func main() {
	r := gin.Default()

	// Libera o navegador (Angular em outra origem) a chamar este mock.
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{frontendOrigin},
		AllowMethods: []string{http.MethodPost, http.MethodOptions},
		AllowHeaders: []string{"Content-Type"},
	}))

	// Responde 405 em JSON quando o método não for o esperado
	r.HandleMethodNotAllowed = true
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"erro": "método não permitido"})
	})

	r.POST("/cobrancas", criarCobranca)
	r.POST("/checkout/:id", registrarResultado)

	log.Printf("[MOCK] Mock de pagamento escutando em %s", addr)
	log.Fatal(r.Run(addr))
}
