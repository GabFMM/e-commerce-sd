// Mock de pagamento (simula o provedor externo) para testes locais.
//
//	POST /cobrancas      {"pedido_id":1,"webhook_url":"..."} -> {"checkout_url":"http://localhost:4200/mock-pagamento/1"}
//	POST /checkout/{id}  {"status":"APROVADO"|"RECUSADO"}    -> repassa o resultado ao webhook do MS Pagamento
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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

func responderJSON(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(corpo)
}

// Libera o navegador (Angular em outra origem) a chamar este mock.
func comCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", frontendOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type cobrancaRequest struct {
	PedidoID   int    `json:"pedido_id"`
	WebhookURL string `json:"webhook_url"`
}

// Chamado pelo MS Pagamento
func criarCobranca(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responderJSON(w, http.StatusMethodNotAllowed, map[string]string{"erro": "método não permitido"})
		return
	}

	var req cobrancaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PedidoID == 0 || req.WebhookURL == "" {
		responderJSON(w, http.StatusBadRequest, map[string]string{"erro": "pedido_id e webhook_url são obrigatórios"})
		return
	}

	mu.Lock()
	cobrancas[req.PedidoID] = req.WebhookURL
	mu.Unlock()

	log.Printf("[MOCK] Cobrança criada para o pedido %d", req.PedidoID)

	responderJSON(w, http.StatusCreated, map[string]string{
		"checkout_url": fmt.Sprintf("%s/%d", checkoutBaseURL, req.PedidoID),
	})
}

type resultadoRequest struct {
	Status string `json:"status"`
}

// Chamado pela página do Angular
func registrarResultado(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responderJSON(w, http.StatusMethodNotAllowed, map[string]string{"erro": "método não permitido"})
		return
	}

	id, err := strconv.Atoi(strings.Trim(strings.TrimPrefix(r.URL.Path, "/checkout/"), "/"))
	if err != nil {
		responderJSON(w, http.StatusBadRequest, map[string]string{"erro": "id inválido"})
		return
	}

	var req resultadoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil ||
		(req.Status != "APROVADO" && req.Status != "RECUSADO") {
		responderJSON(w, http.StatusBadRequest, map[string]string{"erro": "status deve ser APROVADO ou RECUSADO"})
		return
	}

	mu.Lock()
	webhook, existe := cobrancas[id]
	mu.Unlock()
	if !existe {
		responderJSON(w, http.StatusNotFound, map[string]string{"erro": "cobrança não encontrada ou já respondida"})
		return
	}

	corpo, _ := json.Marshal(map[string]any{"pedido_id": id, "status": req.Status})
	resp, err := client.Post(webhook, "application/json", bytes.NewReader(corpo))
	if err != nil {
		log.Printf("[MOCK] Falha ao chamar o webhook do pedido %d: %v", id, err)
		responderJSON(w, http.StatusBadGateway, map[string]string{"erro": "não foi possível chamar o webhook do MS Pagamento"})
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

		responderJSON(w, http.StatusBadGateway, map[string]string{
			"erro": fmt.Sprintf("o MS Pagamento respondeu status %d", resp.StatusCode),
		})
		return
	}

	mu.Lock()
	delete(cobrancas, id)
	mu.Unlock()

	log.Printf("[MOCK] Pedido %d: %s enviado ao webhook", id, req.Status)
	responderJSON(w, http.StatusOK, map[string]string{"mensagem": "ok"})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/cobrancas", criarCobranca)
	mux.HandleFunc("/checkout/", registrarResultado)

	log.Printf("[MOCK] Mock de pagamento escutando em %s", addr)
	log.Fatal(http.ListenAndServe(addr, comCORS(mux)))
}
