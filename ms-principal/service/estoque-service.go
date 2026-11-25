package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"e-commerce-sd/ms-principal/graph/model"
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

const consultaProdutosEstoque = `query Produtos($ids: [Int!], $nomeLike: String) {
	produtos(ids: $ids, nomeLike: $nomeLike) {
		id
		nome
		categoria
		quantidadeDisponivel
	}
}`

type respostaGraphQLEstoque struct {
	Data struct {
		Produtos []*model.Produto `json:"produtos"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// ObterProdutosDoEstoque consulta a API GraphQL do MS Estoque.
// ids == nil omite o filtro; uma slice vazia solicita explicitamente nenhum produto.
func ObterProdutosDoEstoque(ctx context.Context, ids []int32, nomeLike *string) ([]*model.Produto, error) {
	variaveis := make(map[string]any, 2)
	if ids != nil {
		variaveis["ids"] = ids
	}
	if nomeLike != nil {
		variaveis["nomeLike"] = *nomeLike
	}

	corpo, err := json.Marshal(map[string]any{
		"query":     consultaProdutosEstoque,
		"variables": variaveis,
	})
	if err != nil {
		return nil, fmt.Errorf("preparar consulta GraphQL: %w", err)
	}

	endpoint := strings.TrimRight(estoqueURL, "/") + "/graphql"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(corpo))
	if err != nil {
		return nil, fmt.Errorf("%w: preparar requisição: %v", ErrEstoqueIndisponivel, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := estoqueClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEstoqueIndisponivel, err)
	}
	defer resp.Body.Close()

	const limiteResposta = 1 << 20
	corpoResposta, err := io.ReadAll(io.LimitReader(resp.Body, limiteResposta+1))
	if err != nil {
		return nil, fmt.Errorf("%w: erro ao ler resposta: %v", ErrEstoqueIndisponivel, err)
	}
	if len(corpoResposta) > limiteResposta {
		return nil, fmt.Errorf("%w: resposta excede 1 MB", ErrEstoqueIndisponivel)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrEstoqueIndisponivel, resp.StatusCode)
	}

	var resposta respostaGraphQLEstoque
	if err := json.Unmarshal(corpoResposta, &resposta); err != nil {
		return nil, fmt.Errorf("%w: resposta GraphQL inválida: %v", ErrEstoqueIndisponivel, err)
	}
	if len(resposta.Errors) > 0 {
		mensagens := make([]string, 0, len(resposta.Errors))
		for _, erro := range resposta.Errors {
			mensagens = append(mensagens, erro.Message)
		}
		return nil, fmt.Errorf("%w: %s", ErrEstoqueIndisponivel, strings.Join(mensagens, "; "))
	}
	if resposta.Data.Produtos == nil {
		return nil, fmt.Errorf("%w: resposta sem data.produtos", ErrEstoqueIndisponivel)
	}

	return resposta.Data.Produtos, nil
}
