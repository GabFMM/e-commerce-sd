package service

import (
	"e-commerce-sd/data"
	"e-commerce-sd/ms-principal/dto"
	"log"
)

func AtualizarSituacaoPedido(pedidoDTO dto.PedidoDTO, novaSituacao string) {
    ctx, pool := data.ConectarBanco()
    defer pool.Close()

    query := `
        UPDATE pedidos
		SET situacao = $1
		WHERE id = $2
    `

    _, err := pool.Exec(ctx, query, novaSituacao, pedidoDTO.Id)
	if err != nil {
		log.Printf(
            "[ERRO-MS-PRINCIPAL] Não foi possível atualizar a situacao do pedido. pedidoId=%d, novaSituacao=%s: %v",
            pedidoDTO.Id,
			novaSituacao,
            err,
        )
	}

	log.Printf("[INFO-MS-PRINCIPAL] Situação do pedido atualizada. pedidoId=%d, novaSituacao=%s",
		pedidoDTO.Id,
		novaSituacao,
	)
}