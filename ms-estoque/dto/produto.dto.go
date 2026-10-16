package dto

// Produto devolvido pela API REST do estoque.
type ProdutoDTO struct {
	Id                   int    `json:"id"`
	Nome                 string `json:"nome"`
	Categoria            string `json:"categoria"`
	QuantidadeDisponivel int    `json:"quantidadeDisponivel"`
}
