package service

import (
	"database/sql"
	"e-commerce-sd/data"
	"e-commerce-sd/ms-estoque/dto"
	"errors"
	"log"
)

// Retorna true se disponivel, false se indisponivel
// Quem atualiza a situação do pedido é o MS-PRINCIPAL
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

    switch {
    case err == nil:
        return false

    case errors.Is(err, sql.ErrNoRows):
        return true

    default:
        log.Printf(
            "[ERRO] Não foi possível verificar a disponibilidade do pedido. PedidoId=%d: %v",
            pedidoDTO.Id,
            err,
        )
        return false
    }
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
