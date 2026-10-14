package service

import (
	"e-commerce-sd/data"
	"e-commerce-sd/ms-estoque/dto"
)

// Retorna os produtos com saldo disponível (quantidade - reservados > 0).
// Ajuste as colunas do SELECT ao seu schema (copie do antigo service.ObterProdutos do MS Principal).
func ListarProdutosDisponiveis() ([]dto.ProdutoDTO, error) {
	ctx, pool := data.ConectarBanco()
	// Removido 'defer pool.Close()' para não encerrar o pool da aplicação

	query := `
		SELECT id, nome, quantidade,(quantidade - reservados) AS disponivel
		FROM produtos
		WHERE (quantidade - reservados) > 0
		ORDER BY id;
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	produtos := make([]dto.ProdutoDTO, 0)
	for rows.Next() {
		var p dto.ProdutoDTO
		if err := rows.Scan(&p.Id, &p.Nome, &p.Quantidade, &p.Disponivel); err != nil {
			return nil, err
		}
		produtos = append(produtos, p)
	}

	return produtos, rows.Err()
}
