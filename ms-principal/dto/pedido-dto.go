package dto

// PedidoDTO é o payload simples (só o id) dos eventos de status e de pedido.excluido.
type PedidoDTO struct {
	Id int `json:"pedido_id"`
}

type ItemDTO struct {
	ProdutoId  int `json:"produto_id"`
	Quantidade int `json:"quantidade"`
}

// PedidoCriadoDTO é o payload de pedido.criado. Leva os itens porque o MS Estoque
// tem banco próprio e não enxerga as tabelas do MS Principal.
type PedidoCriadoDTO struct {
	Id    int       `json:"pedido_id"`
	Itens []ItemDTO `json:"itens"`
}
