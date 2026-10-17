package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ErrEstoqueIndisponivel é devolvido quando o MS Estoque não responde ou responde com erro.
var ErrEstoqueIndisponivel = errors.New("ms estoque indisponível")

// URL base do MS Estoque (troque com a variável de ambiente ESTOQUE_URL).
var estoqueURL = func() string {
	if v := os.Getenv("ESTOQUE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8081"
}()

var estoqueClient = &http.Client{Timeout: 5 * time.Second}

// ObterProdutosDoEstoque consulta GET /produtos no MS Estoque (via REST) e devolve o JSON
// como veio. O gateway só repassa a lista, então não depende da estrutura dos produtos.
func ObterProdutosDoEstoque(query string) (json.RawMessage, error) {

	// Constrói o endpoint com os parâmetros recebidos
	endpoint := estoqueURL + "/produtos"
	if query != "" {
		endpoint += "?" + query
	}

	// Realiza a requisição
	resp, err := estoqueClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEstoqueIndisponivel, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrEstoqueIndisponivel, resp.StatusCode)
	}

	corpo, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // até 1 MB
	if err != nil {
		return nil, fmt.Errorf("%w: erro ao ler resposta: %v", ErrEstoqueIndisponivel, err)
	}

	if !json.Valid(corpo) {
		return nil, fmt.Errorf("%w: resposta não é um JSON válido", ErrEstoqueIndisponivel)
	}

	return corpo, nil
}
