package service

import (
	"e-commerce-sd/data"
	"e-commerce-sd/ms-principal/dto"
	"log"
)

type Produto struct {
	Id int
	Nome string
	Categoria string
	QuantidadeDisponivel int
}

type Pedido struct {
	Id int
	Situacao string

	// Chave corresponde ao nome do produto
	// Valor corresponde a quantidade requisitada
	Produtos map[string]int
}

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

func ObterPedidos() []Pedido {
    ctx, pool := data.ConectarBanco()
    defer pool.Close()

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
            id           int
            situacao     string
            nomeProduto  string
            quantidade   int
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

func ObterProdutos() []Produto {
	ctx, pool := data.ConectarBanco()
    defer pool.Close()

	query := `
		SELECT 
			id, 
			nome, 
			categoria, 
			quantidade - reservados AS quantidadeDisponivel
		FROM produtos
	`

	rows, err := pool.Query(ctx, query)
    if err != nil {
        log.Printf("[ERRO-MS-PRINCIPAL] Não foi possível recuperar os pedidos do usuário: %v", err)
        return nil
    }
    defer rows.Close()

	var produtos []Produto

	for rows.Next() {
		var produto Produto

		if err := rows.Scan(
            &produto.Id,
            &produto.Nome,
            &produto.Categoria,
            &produto.QuantidadeDisponivel,
        ); err != nil {
            log.Printf(
                "[ERRO-MS-PRINCIPAL] Falha ao converter registros do banco de dados para struct Produto: %v",
                err,
            )
            continue
        }

		produtos = append(produtos, produto)
	}

	if err := rows.Err(); err != nil {
        log.Printf("[ERRO-MS-PRINCIPAL] Erro ao percorrer registros dos produtos: %v", err)
        return nil
    }

	return produtos
}

// Retorna o ID do novo pedido e true, se a criação foi um sucesso.
// Retorna 0 e false, caso contrário.
func CriarPedido() (int, bool) {
	ctx, pool := data.ConectarBanco()
	defer pool.Close()

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
	ctx, pool := data.ConectarBanco()
	defer pool.Close()

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
	ctx, pool := data.ConectarBanco()
	defer pool.Close()

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
	ctx, pool := data.ConectarBanco()
	defer pool.Close()

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
	ctx, pool := data.ConectarBanco()
	defer pool.Close()

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
		return nil
	}
	defer rows.Close()

	var pedidos []Pedido

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