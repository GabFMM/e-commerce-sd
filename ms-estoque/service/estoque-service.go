package service

import (
	"context"
	"e-commerce-sd/data"
	"e-commerce-sd/ms-estoque/dto"
	"log"
)

// Retorna true se disponivel, false se indisponivel
// Quem atualiza a situação do pedido é o MS-PRINCIPAL
func VerificarDisponibilidade(pedidoDTO dto.PedidoDTO) bool {
	ctx := context.Background()
	pool := data.Pool("ESTOQUE")

	// EXISTS devolve sempre uma linha com um booleano: true se algum item do pedido
	// pede mais do que o saldo disponível (quantidade - reservados).
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM produtos p
			INNER JOIN pedidos_produtos pp
				ON pp.produto_id = p.id
			WHERE pp.pedido_id = $1
			  AND (p.quantidade - p.reservados) < pp.quantidade
		);
	`

	var faltaEstoque bool
	if err := pool.QueryRow(ctx, query, pedidoDTO.Id).Scan(&faltaEstoque); err != nil {
		log.Printf(
			"[ERRO] Não foi possível verificar a disponibilidade do pedido. PedidoId=%d: %v",
			pedidoDTO.Id,
			err,
		)
		return false
	}

	return !faltaEstoque
}

func ReservarProdutos(pedidoDTO dto.PedidoDTO) {
	ctx := context.Background()
	pool := data.Pool("ESTOQUE")

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
	ctx := context.Background()
	pool := data.Pool("ESTOQUE")

	// GREATEST evita que "reservados" fique negativo caso o pedido nunca tenha reservado nada.
	query := `
		UPDATE produtos p
		SET reservados = GREATEST(p.reservados - pp.quantidade, 0)
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
