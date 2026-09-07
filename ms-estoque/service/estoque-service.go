package service

import (
	"database/sql"
	"e-commerce-sd/data"
	"e-commerce-sd/ms-estoque/dto"
	"errors"
	"log"
)

func VerificarDisponibilidade(pedidoDTO dto.PedidoDTO) bool {
    ctx, pool := data.ConectarBanco()
    defer pool.Close()

    query := `
        SELECT 1
        FROM produtos p
        INNER JOIN pedidos_produtos pp
            ON pp.produto_id = p.id
        WHERE pp.pedido_id = $1
          AND (p.quantidade - p.reservados) < pp.quantidade
        LIMIT 1;
    `

    row := pool.QueryRow(ctx, query, pedidoDTO.Id)

    err := row.Scan()

    var situacao string

    switch {
    case err == nil:
        situacao = "ESTOQUE_INDISPONIVEL"

    case errors.Is(err, sql.ErrNoRows):
        situacao = "ESTOQUE_DISPONIVEL"

    default:
        log.Printf(
            "[ERRO] Não foi possível verificar a disponibilidade do pedido. PedidoId=%d: %v",
            pedidoDTO.Id,
            err,
        )
        return false
    }

    query = `
        UPDATE pedidos
        SET situacao = $1
        WHERE id = $2
    `

    _, err = pool.Exec(ctx, query, situacao, pedidoDTO.Id)
    if err != nil {
        log.Printf(
            "[ERRO] Não foi possível atualizar a situação do pedido. PedidoId=%d: %v",
            pedidoDTO.Id,
            err,
        )
        return false
    }

    log.Printf(
        "[INFO] Situação do pedido mudada para %s. PedidoId=%d",
        situacao,
        pedidoDTO.Id,
    )

    return true
}

func ReservarProdutos(pedidoDTO dto.PedidoDTO) {
	ctx, pool := data.ConectarBanco()
	defer pool.Close()

	query := `
		UPDATE produtos p
		SET reservados = p.reservados + pp.quantidade
		FROM pedidos_produtos pp
		WHERE pp.pedido_id = $1
		  AND pp.produto_id = p.id;
	`

	_, err := pool.Exec(ctx, query, pedidoDTO.Id)
	if err != nil {
		log.Printf(
			"[ERRO] Não foi possível reservar os produtos. PedidoId=%d: %v",
			pedidoDTO.Id,
			err,
		)
		return
	}

	log.Printf(
		"[INFO] Produtos reservados com sucesso. PedidoId=%d",
		pedidoDTO.Id,
	)
}

func RemoverReservas(pedidoDTO dto.PedidoDTO) {
	ctx, pool := data.ConectarBanco()
	defer pool.Close()

	query := `
		UPDATE produtos p
		SET reservados = p.reservados - pp.quantidade
		FROM pedidos_produtos pp
		WHERE pp.pedido_id = $1
		  AND pp.produto_id = p.id;
	`

	_, err := pool.Exec(ctx, query, pedidoDTO.Id)
	if err != nil {
		log.Printf(
			"[ERRO] Não foi possível remover as reservas. PedidoId=%d: %v",
			pedidoDTO.Id,
			err,
		)
		return
	}

	log.Printf(
		"[INFO] Reservas removidas com sucesso. PedidoId=%d",
		pedidoDTO.Id,
	)
}
