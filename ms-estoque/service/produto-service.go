package service

import (
	"context"
	"e-commerce-sd/data"
	"e-commerce-sd/ms-estoque/graph/model"
	"fmt"
)

func ObterProdutosFiltros(nomeLike string, nomeLikePresente bool, ids []int32, idsPresentes bool) ([]*model.Produto, error) {

	ctx := context.Background()
	pool := data.Pool("ESTOQUE")

	// Query básica
	query := `
		SELECT
			id,
			nome,
			categoria,
			(quantidade - reservados) AS disponivel
		FROM produtos
		WHERE (quantidade - reservados) > 0
	`

	// Tratamento dos parâmetros opcionais
	args := make([]any, 0, 2)

	if nomeLikePresente {
		args = append(args, nomeLike)
		query += fmt.Sprintf(" AND nome ILIKE '%%' || $%d || '%%'", len(args))
	}

	if idsPresentes {
		if len(ids) == 0 {
			return []*model.Produto{}, nil
		}

		args = append(args, ids)
		query += fmt.Sprintf(" AND id = ANY($%d)", len(args))
	}

	query += " ORDER BY id"

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	produtos := make([]*model.Produto, 0)

	for rows.Next() {
		var p model.Produto

		if err := rows.Scan(
			&p.ID,
			&p.Nome,
			&p.Categoria,
			&p.QuantidadeDisponivel,
		); err != nil {
			return nil, err
		}

		produtos = append(produtos, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return produtos, nil
}
