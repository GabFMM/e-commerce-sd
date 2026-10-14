package service

import (
	"context"
	"e-commerce-sd/data"
	"e-commerce-sd/ms-estoque/dto"
	"errors"
	"fmt"
	"log"
)

// Devolvido quando o evento pedido.criado chega duas vezes para o mesmo pedido.
var ErrPedidoJaProcessado = errors.New("pedido já processado pelo estoque")

// ProcessarPedidoCriado registra o pedido no banco do estoque, verifica a
// disponibilidade e, se houver saldo para TODOS os itens, reserva tudo.
func ProcessarPedidoCriado(pedido dto.PedidoCriadoDTO) (bool, error) {
	if len(pedido.Itens) == 0 {
		return false, nil
	}

	ctx := context.Background()
	pool := data.Pool("ESTOQUE")

	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Correção: Especificar situacao 'CRIADO' para respeitar a constraint NOT NULL
	res, err := tx.Exec(ctx,
		`INSERT INTO pedidos (id, situacao) VALUES ($1, 'CRIADO') ON CONFLICT (id) DO NOTHING`, pedido.Id)
	if err != nil {
		return false, fmt.Errorf("registrar pedido: %w", err)
	}
	if res.RowsAffected() == 0 {
		return false, ErrPedidoJaProcessado
	}

	for _, item := range pedido.Itens {
		_, err := tx.Exec(ctx, `
			INSERT INTO pedidos_produtos (pedido_id, produto_id, quantidade)
			VALUES ($1, $2, $3)
		`, pedido.Id, item.ProdutoId, item.Quantidade)
		if err != nil {
			return false, fmt.Errorf("registrar item (produto %d): %w", item.ProdutoId, err)
		}
	}

	// Trava as linhas dos produtos do pedido (ORDER BY evita deadlock entre pedidos)
	_, err = tx.Exec(ctx, `
		SELECT 1 FROM produtos
		WHERE id IN (SELECT produto_id FROM pedidos_produtos WHERE pedido_id = $1)
		ORDER BY id
		FOR UPDATE
	`, pedido.Id)
	if err != nil {
		return false, fmt.Errorf("travar produtos: %w", err)
	}

	// true se algum item pede mais do que o saldo disponível (quantidade - reservados)
	var faltaEstoque bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM produtos p
			INNER JOIN pedidos_produtos pp ON pp.produto_id = p.id
			WHERE pp.pedido_id = $1
			  AND (p.quantidade - p.reservados) < pp.quantidade
		)
	`, pedido.Id).Scan(&faltaEstoque)
	if err != nil {
		return false, fmt.Errorf("verificar disponibilidade: %w", err)
	}
	if faltaEstoque {
		log.Printf("[INFO-MS-ESTOQUE] Estoque insuficiente. PedidoId=%d", pedido.Id)
		return false, nil // O defer faz Rollback
	}

	_, err = tx.Exec(ctx, `
		UPDATE produtos p
		SET reservados = p.reservados + pp.quantidade
		FROM pedidos_produtos pp
		WHERE pp.pedido_id = $1
		  AND pp.produto_id = p.id
	`, pedido.Id)
	if err != nil {
		return false, fmt.Errorf("reservar produtos: %w", err)
	}

	// 2. Correção: Atualizar a coluna situacao para 'ESTOQUE_DISPONIVEL' no lugar da coluna reservado inexistente
	_, err = tx.Exec(ctx, `UPDATE pedidos SET situacao = 'ESTOQUE_DISPONIVEL' WHERE id = $1`, pedido.Id)
	if err != nil {
		return false, fmt.Errorf("marcar pedido como reservado: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}

	log.Printf("[INFO-MS-ESTOQUE] Produtos reservados com sucesso. PedidoId=%d", pedido.Id)
	return true, nil
}

// RemoverReservas devolve ao saldo o que o pedido tinha reservado e apaga o registro local.
func RemoverReservas(pedidoDTO dto.PedidoDTO) error {
	ctx := context.Background()
	pool := data.Pool("ESTOQUE")

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback(ctx)

	// 3. Correção: Verificar pe.situacao = 'ESTOQUE_DISPONIVEL' em vez de pe.reservado
	_, err = tx.Exec(ctx, `
		UPDATE produtos p
		SET reservados = p.reservados - pp.quantidade
		FROM pedidos_produtos pp
		INNER JOIN pedidos pe ON pe.id = pp.pedido_id
		WHERE pp.pedido_id = $1
		  AND pe.situacao = 'ESTOQUE_DISPONIVEL'
		  AND pp.produto_id = p.id
	`, pedidoDTO.Id)
	if err != nil {
		return fmt.Errorf("devolver reservas: %w", err)
	}

	// ON DELETE CASCADE apaga os pedidos_produtos
	_, err = tx.Exec(ctx, `DELETE FROM pedidos WHERE id = $1`, pedidoDTO.Id)
	if err != nil {
		return fmt.Errorf("remover pedido: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.Printf("[INFO-MS-ESTOQUE] Reservas removidas. PedidoId=%d", pedidoDTO.Id)
	return nil
}
