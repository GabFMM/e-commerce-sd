package dto

// Produto devolvido pela API REST do estoque.
// Ajuste os campos ao que o frontend já consome (copie do antigo service.ObterProdutos do MS Principal).
type ProdutoDTO struct {
	Id         int    `json:"id"`
	Nome       string `json:"nome"`
	Quantidade int    `json:"quantidade"` // disponível = quantidade - reservados
	Disponivel int    `json:"disponivel"`
}
