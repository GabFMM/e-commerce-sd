package service

import (
	"context"
	"e-commerce-sd/data"
	"e-commerce-sd/ms-principal/dto"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgconn"
)

type Pedido struct {
	Id       int
	Situacao string

	// Chave corresponde ao nome do produto
	// Valor corresponde a quantidade requisitada
	Produtos map[string]int
}

func AtualizarSituacaoPedido(pedidoDTO dto.PedidoDTO, novaSituacao string) {
	ctx := context.Background()
	pool := data.Pool("PRINCIPAL")

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

func ObterPedidos() []Pedido {
	ctx := context.Background()
	pool := data.Pool("PRINCIPAL")

	query := `
        SELECT 
            pedidos.id,
            pedidos.situacao,
            produtos.nome,
            pedidos_produtos.quantidade
        FROM pedidos
        INNER JOIN pedidos_produtos
            ON pedidos_produtos.pedido_id = pedidos.id
        INNER JOIN produtos
            ON produtos.id = pedidos_produtos.produto_id
    `

	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Printf("[ERRO-MS-PRINCIPAL] Não foi possível recuperar os pedidos do usuário: %v", err)
		return nil
	}
	defer rows.Close()

	pedidosMap := make(map[int]*Pedido)

	for rows.Next() {

		// Representa a relação de um pedido para um produto
		var (
			id          int
			situacao    string
			nomeProduto string
			quantidade  int
		)

		if err := rows.Scan(
			&id,
			&situacao,
			&nomeProduto,
			&quantidade,
		); err != nil {
			log.Printf(
				"[ERRO-MS-PRINCIPAL] Falha ao converter registros do banco de dados para struct Pedido: %v",
				err,
			)
			continue
		}

		pedido, existe := pedidosMap[id]

		if !existe {
			pedido = &Pedido{
				Id:       id,
				Situacao: situacao,
				Produtos: make(map[string]int),
			}

			pedidosMap[id] = pedido
		}

		pedido.Produtos[nomeProduto] = quantidade
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERRO-MS-PRINCIPAL] Erro ao percorrer registros dos pedidos: %v", err)
		return nil
	}

	pedidos := make([]Pedido, 0, len(pedidosMap))

	for _, pedido := range pedidosMap {
		pedidos = append(pedidos, *pedido)
	}

	log.Printf("[INFO-MS-PRINCIPAL] Pedidos do usuário recuperados do banco de dados")

	return pedidos
}

// Retorna o ID do novo pedido e true, se a criação foi um sucesso.
// Retorna 0 e false, caso contrário.
func CriarPedido() (int, bool) {
	ctx := context.Background()
	pool := data.Pool("PRINCIPAL")

	query := `
		INSERT INTO pedidos (situacao)
		VALUES ('CRIADO')
		RETURNING id
	`

	var id int

	err := pool.QueryRow(ctx, query).Scan(&id)
	if err != nil {
		log.Printf(
			"[ERRO-MS-PRINCIPAL] Não foi possível criar o pedido: %v",
			err,
		)
		return 0, false
	}

	log.Printf("[INFO-MS-PRINCIPAL] Pedido %d criado", id)
	return id, true
}

func AdicionarProdutoPedido(pedidoId int, produtoId int, quantidade int) bool {
	ctx := context.Background()
	pool := data.Pool("PRINCIPAL")

	query := `
		INSERT INTO pedidos_produtos (pedido_id, produto_id, quantidade)
		VALUES ($1, $2, $3)
	`

	_, err := pool.Exec(ctx, query, pedidoId, produtoId, quantidade)
	if err != nil {
		log.Printf(
			"[ERRO-MS-PRINCIPAL] Não foi possível adicionar o produto no pedido. PedidoId=%d, ProdutoId=%d: %v",
			pedidoId,
			produtoId,
			err,
		)
		return false
	}

	log.Printf("[INFO-MS-PRINCIPAL] Produto %d adicionado no Pedido %d", produtoId, pedidoId)
	return true
}

func ExcluirPedido(pedidoId int) bool {
	ctx := context.Background()
	pool := data.Pool("PRINCIPAL")

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Printf(
			"[ERRO-MS-PRINCIPAL] Não foi possível iniciar a transação. PedidoId=%d: %v",
			pedidoId,
			err,
		)
		return false
	}
	defer tx.Rollback(ctx)

	// Devolve ao estoque os produtos reservados pelo pedido.
	queryAtualizarProdutos := `
		UPDATE produtos
		SET reservados = reservados - pp.quantidade
		FROM pedidos_produtos pp
		WHERE produtos.id = pp.produto_id
		AND pp.pedido_id = $1
	`

	_, err = tx.Exec(ctx, queryAtualizarProdutos, pedidoId)
	if err != nil {
		log.Printf(
			"[ERRO-MS-PRINCIPAL] Não foi possível atualizar as reservas dos produtos. PedidoId=%d: %v",
			pedidoId,
			err,
		)
		return false
	}

	// Remove o pedido.
	queryExcluirPedido := `
		DELETE FROM pedidos
		WHERE id = $1
	`

	resultado, err := tx.Exec(ctx, queryExcluirPedido, pedidoId)
	if err != nil {
		log.Printf(
			"[ERRO-MS-PRINCIPAL] Não foi possível remover o pedido. PedidoId=%d: %v",
			pedidoId,
			err,
		)
		return false
	}

	if resultado.RowsAffected() == 0 {
		log.Printf(
			"[ERRO-MS-PRINCIPAL] Pedido não encontrado. PedidoId=%d",
			pedidoId,
		)
		return false
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf(
			"[ERRO-MS-PRINCIPAL] Não foi possível confirmar a exclusão do pedido. PedidoId=%d: %v",
			pedidoId,
			err,
		)
		return false
	}

	log.Printf("[INFO-MS-PRINCIPAL] Pedido %d removido", pedidoId)
	return true
}

func ObterPedido(pedidoId int) *Pedido {
	ctx := context.Background()
	pool := data.Pool("PRINCIPAL")

	query := `
		SELECT id, situacao
		FROM pedidos
		WHERE id = $1
	`

	var pedido Pedido

	err := pool.QueryRow(ctx, query, pedidoId).Scan(
		&pedido.Id,
		&pedido.Situacao,
	)

	if err != nil {
		log.Printf(
			"[ERRO-MS-PRINCIPAL] Não foi possível consultar o pedido. PedidoId=%d: %v",
			pedidoId,
			err,
		)
		return nil
	}

	return &pedido
}

func ObterPedidosPorSituacao(situacao string) []Pedido {
	ctx := context.Background()
	pool := data.Pool("PRINCIPAL")

	query := `
        SELECT id, situacao
        FROM pedidos
        WHERE situacao = $1
        ORDER BY id
    `

	rows, err := pool.Query(ctx, query, situacao)
	if err != nil {
		log.Printf(
			"[ERRO-MS-PRINCIPAL] Não foi possível consultar pedidos. Situacao=%s: %v",
			situacao,
			err,
		)
		return nil // Retorna nil apenas em caso de erro real de banco
	}
	defer rows.Close()

	// Inicializa o slice como vazio para evitar retornar nil quando não houver resultados
	pedidos := []Pedido{}

	for rows.Next() {
		var pedido Pedido

		err := rows.Scan(
			&pedido.Id,
			&pedido.Situacao,
		)

		if err != nil {
			log.Printf(
				"[ERRO-MS-PRINCIPAL] Não foi possível ler pedido. %v",
				err,
			)
			return nil
		}

		pedidos = append(pedidos, pedido)
	}

	return pedidos
}

var (
	ErrPedidoSemItens     = errors.New("pedido sem itens")
	ErrProdutoInexistente = errors.New("produto inexistente")
	ErrItemDuplicado      = errors.New("produto repetido no pedido")
)

type ItemPedido struct {
	ProdutoId  int
	Quantidade int
}

func CriarPedidoComItens(itens []ItemPedido) (int, error) {
	if len(itens) == 0 {
		return 0, ErrPedidoSemItens
	}

	ctx := context.Background()
	pool := data.Pool("PRINCIPAL")

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Printf("[ERRO-MS-PRINCIPAL] Não foi possível iniciar a transação do pedido: %v", err)
		return 0, fmt.Errorf("iniciar transação: %w", err)
	}
	// Após o Commit, o Rollback vira no-op (retorna ErrTxClosed, ignorado).
	defer tx.Rollback(ctx)

	// 1. Cria o pedido
	var pedidoId int

	err = tx.QueryRow(ctx, `
		INSERT INTO pedidos (situacao)
		VALUES ('CRIADO')
		RETURNING id
	`).Scan(&pedidoId)
	if err != nil {
		log.Printf("[ERRO-MS-PRINCIPAL] Não foi possível criar o pedido: %v", err)
		return 0, fmt.Errorf("criar pedido: %w", err)
	}

	// 2. Adiciona cada item
	for _, item := range itens {
		_, err = tx.Exec(ctx, `
			INSERT INTO pedidos_produtos (pedido_id, produto_id, quantidade)
			VALUES ($1, $2, $3)
		`, pedidoId, item.ProdutoId, item.Quantidade)

		if err != nil {
			log.Printf(
				"[ERRO-MS-PRINCIPAL] Não foi possível adicionar o produto no pedido. PedidoId=%d, ProdutoId=%d: %v",
				pedidoId,
				item.ProdutoId,
				err,
			)

			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				switch pgErr.Code {
				case "23503": // foreign_key_violation
					return 0, fmt.Errorf("%w: id=%d", ErrProdutoInexistente, item.ProdutoId)
				case "23505": // unique_violation
					return 0, fmt.Errorf("%w: id=%d", ErrItemDuplicado, item.ProdutoId)
				}
			}

			return 0, fmt.Errorf("adicionar item: %w", err)
		}
	}

	// 3. Confirma tudo de uma vez
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[ERRO-MS-PRINCIPAL] Não foi possível confirmar a criação do pedido: %v", err)
		return 0, fmt.Errorf("commit: %w", err)
	}

	log.Printf("[INFO-MS-PRINCIPAL] Pedido %d criado com %d item(ns)", pedidoId, len(itens))
	return pedidoId, nil
}
