package dto

type PedidoDTO struct {
	Id int `json:"pedido_id"`
}

type ItemDTO struct {
	ProdutoId  int `json:"produto_id"`
	Quantidade int `json:"quantidade"`
}

type PedidoCriadoDTO struct {
	Id    int       `json:"pedido_id"`
	Itens []ItemDTO `json:"itens"`
}
