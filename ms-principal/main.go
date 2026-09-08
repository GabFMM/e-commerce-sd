package main

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"e-commerce-sd/ms-principal/dto"
	"e-commerce-sd/ms-principal/service"
	"e-commerce-sd/seguranca"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeExchange = "eCommerce"
const NomeServico = "principal"
const NomeFila = "principal"

// Encerra o MS
func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

// Não encerra o MS
func printOnError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "[ERRO-MS-PRINCIPAL] Erro ao conectar ao RabbitMQ")
	defer conn.Close()

	// Canal dedicado para publicar e canal dedicado para consumir
	publishCh, err := conn.Channel()
	failOnError(err, "[ERRO-MS-PRINCIPAL] Erro ao abrir channel de publish")
	defer publishCh.Close()

	consumeCh, err := conn.Channel()
	failOnError(err, "[ERRO-MS-PRINCIPAL] Erro ao abrir channel de consume")
	defer consumeCh.Close()

	// Liga-se na fila de consumidores
	msgs, err := consumeCh.Consume(
		NomeFila, // queue
		"",       // id do consumidor (vazio para o rabbitmq gerar automaticamente)
		true,     // auto-ack
		true,     // exclusive (apenas o ms-estoque lê a fila dele)
		false,    // no-local
		false,    // no-wait
		nil,      // args
	)
	failOnError(err, "[ERRO-MS-PRINCIPAL] Falha ao registrar na fila do consumidor")

	chavesPublicas, err := seguranca.CarregarTodasChavesPublicas("ms-estoque/")
	failOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível carregar as chaves públicas")

	chavePrivada, err := seguranca.CarregarChavePrivada("chaves/estoque_private.pem")
	failOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível carregar chave privada")

	var forever chan struct{}

	// Processa mensagens da fila
	go func() {
		for d := range msgs {
			payload, err := seguranca.AbrirPacote(d.Body, chavesPublicas)

			printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível descriptografar/autenticar/interpretar pacote recebido")
			if err != nil {
				continue
			}

			switch d.RoutingKey {
			case "pagamento.aprovado":
				pagamentoAprovado(payload)
			case "pagamento.recusado":
				pagamentoRecusado(publishCh, chavePrivada, payload)
			case "pedido.enviado":
				pedidoEnviado(payload)
			case "pedido.estoque_ok":
				estoqueOk(payload)
			case "estoque.indisponivel":
				estoqueIndisponivel(publishCh, chavePrivada, payload)
			default:
				printOnError(err, "[ERRO-MS-PRINCIPAL] routing key inválida:"+d.RoutingKey)
			}
		}
	}()

	// Processa entradas do usuário, junto da exibição do menu
	go func () {
		criarMenu(publishCh, chavePrivada)
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func pagamentoAprovado(payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "PAGAMENTO_APROVADO")
	// Não publica nada
}

func pagamentoRecusado(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "PAGAMENTO_RECUSADO")
	
	publicar(publishCh, chavePrivada, "pedido.excluido", pedidoDTO)
}

func pedidoEnviado(payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "ENVIADO")
	// Não publica nada
}

func estoqueOk(payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "ESTOQUE_DISPONIVEL")
	// Não publica nada
}

func estoqueIndisponivel(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "ESTOQUE_INDISPONIVEL")
	
	publicar(publishCh, chavePrivada, "pedido.excluido", pedidoDTO)
}

func publicar(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, routingKey string, body interface{}) {
	bodyByte, err := seguranca.CriarPacote(body, NomeServico, chavePrivada)
	if err != nil {
		log.Printf("[ERRO-MS-PRINCIPAL] Não foi possível criar o pacote para %s: %v", routingKey, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = publishCh.PublishWithContext(ctx,
		NomeExchange,        // exchange
		routingKey,          // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        bodyByte,
		},
	)

	if err != nil {
		log.Printf("[ERRO-MS-PRINCIPAL] Falha ao publicar pedido.estoque_ok: %v", err)
		return
	}

	log.Printf("[INFO-MS-PRINCIPAL] Pacote publicado para %s", routingKey)
}

func lerOpcaoEscolhidaMinMax(min int, max int) int {
	var opcao int

	_, err := fmt.Scan(&opcao)
	for err != nil || opcao < min || opcao > max {
		fmt.Printf("Entrada inválida. Tente novamente")

		if err != nil {
			// Descarta a linha com o texto inválido
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				_ = scanner.Text() // Consome o texto errado para liberar o buffer
			}
		}

		fmt.Printf("Entrada inválida. Tente novamente:")
		_, err = fmt.Scan(&opcao) 
	}

	return opcao
}

func lerOpcaoEscolhidaListaInt(lista []int) int {
	var opcao int

	_, err := fmt.Scan(&opcao)
	for err != nil || !slices.Contains(lista, opcao) {
		fmt.Printf("Entrada inválida. Tente novamente")

		if err != nil {
			// Descarta a linha com o texto inválido
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				_ = scanner.Text() // Consome o texto errado para liberar o buffer
			}
		}

		fmt.Print("Entrada inválida. Tente novamente:")
		_, err = fmt.Scan(&opcao) 
	}

	return opcao
}

func lerOpcaoEscolhidaListaStr(lista []string) string {
	var opcao string

	_, err := fmt.Scan(&opcao)
	for err != nil || !slices.Contains(lista, opcao) {
		fmt.Printf("Entrada inválida. Tente novamente")

		if err != nil {
			// Descarta a linha com o texto inválido
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				_ = scanner.Text() // Consome o texto errado para liberar o buffer
			}
		}

		fmt.Print("Entrada inválida. Tente novamente:")
		_, err = fmt.Scan(&opcao) 
	}

	return opcao
}

func criarMenu(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey) {
	fmt.Println(">> MENU PRINCIPAL <<")
	fmt.Println("")
	fmt.Println("Digite o número de uma opção:")
	fmt.Println("1 - Visualizar pedidos")
	fmt.Println("2 - Realizar pedidos")
	fmt.Println("3 - Excluir pedidos")
	fmt.Println("4 - Consultar pedidos e respectivos status")
	fmt.Println("")

	opcao := lerOpcaoEscolhidaMinMax(1, 4)

	switch opcao {
	case 1:
		visualizarPedidos(publishCh, chavePrivada)
	case 2:
		realizarPedidos(publishCh, chavePrivada)
	case 3:
		excluirPedidos(publishCh, chavePrivada)
	case 4:
		consultarPedidos(publishCh, chavePrivada)
	}
}

func visualizarPedidos(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey) {
	pedidos := service.ObterPedidos()

	if pedidos == nil {
		fmt.Println("Não foi possível recuperar seus pedidos")
		fmt.Println("")
	} else {
		for _, pedido := range pedidos {
			fmt.Printf("PEDIDO %d:", pedido.Id)
			fmt.Printf("- Situacao: %s", pedido.Situacao)
			fmt.Println("- Pedidos:")

			for chave, valor := range pedido.Produtos {
				fmt.Printf("\t=> %s (%d)", chave, valor)
			}

			fmt.Println("")
		}
	}

	fmt.Println("Digite 1 para voltar ao menu principal")
	lerOpcaoEscolhidaMinMax(1, 1)

	criarMenu(publishCh, chavePrivada)
}

func realizarPedidos(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey) {
	pedidoId, criado := service.CriarPedido()

	if !criado {
		fmt.Println("Não foi possível criar seu pedido")
		fmt.Println("")
		fmt.Println("Digite 1 para voltar ao menu principal")
		lerOpcaoEscolhidaMinMax(1, 1)

		criarMenu(publishCh, chavePrivada)
	}

	produtos := service.ObterProdutos()

	if produtos == nil {
		fmt.Println("Não há produtos cadastrados")
		fmt.Println("")
		fmt.Println("Digite 1 para voltar ao menu principal")
		lerOpcaoEscolhidaMinMax(1, 1)

		criarMenu(publishCh, chavePrivada)
	} else {
		resposta := "S"

		for strings.ToUpper(resposta) == "S" {
			var ids []int

			for _, produto := range produtos {
				ids = append(ids, produto.Id)

				fmt.Printf("%s:", produto.Nome)
				fmt.Printf("- ID: %d", produto.Id)
				fmt.Printf("- Categoria: %s", produto.Categoria)
				fmt.Printf("- Quantidade disponível: %d", produto.QuantidadeDisponivel)
				fmt.Println("")
			}

			// -- ID do produto
			fmt.Println("Digite o ID do produto a ser adicionado no pedido:")
			fmt.Println("")

			produtoId := lerOpcaoEscolhidaListaInt(ids)
			// --

			// -- Quantidade do produto
			var quantidadeMax int

			for _, produto := range produtos {
				if produto.Id == produtoId {
					quantidadeMax = produto.QuantidadeDisponivel
					break
				}
			}

			fmt.Println("Digite a quantidade desejada do produto:")
			fmt.Println("")

			quantidade := lerOpcaoEscolhidaMinMax(1, quantidadeMax)
			// --

			if !service.AdicionarProdutoPedido(pedidoId, produtoId, quantidade) {
				fmt.Println("Não foi possível adicionar o produto ao pedido.")
				fmt.Println("")
			} else {
				fmt.Println("Sucesso. Produto adicionado ao pedido.")
				fmt.Println("")

				var pedido dto.PedidoDTO
				pedido.Id = pedidoId

				publicar(publishCh, chavePrivada, "pedido.criado", pedido)
			}

			fmt.Println("Deseja adicionar outro produto ao mesmo pedido? (S/N)")
			resposta = lerOpcaoEscolhidaListaStr([]string{"S", "N", "s", "n"})

			fmt.Println("")
		}

		fmt.Println("Digite 1 para voltar ao menu principal")
		lerOpcaoEscolhidaMinMax(1, 1)

		criarMenu(publishCh, chavePrivada)
	}
}

func excluirPedidos(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey) {
	fmt.Println("Deseja visualizar os pedidos antes de excluir? (S/N)")
	respostaVisualizar := lerOpcaoEscolhidaListaStr([]string{"S", "N", "s", "n"})

	if strings.ToUpper(respostaVisualizar) == "S" {
		pedidos := service.ObterPedidos()

		if pedidos == nil {
			fmt.Println("Não foi possível recuperar seus pedidos")
			fmt.Println("")
		} else if len(pedidos) == 0 {
			fmt.Println("Não há pedidos cadastrados")
			fmt.Println("")
		} else {
			for _, pedido := range pedidos {
				fmt.Printf("PEDIDO %d:", pedido.Id)
				fmt.Printf("- Situacao: %s", pedido.Situacao)
				fmt.Println("- Pedidos:")

				for chave, valor := range pedido.Produtos {
					fmt.Printf("\t=> %s (%d)", chave, valor)
				}

				fmt.Println("")
			}
		}
	}

	resposta := "S"

	for strings.ToUpper(resposta) == "S" {
		pedidos := service.ObterPedidos()

		if pedidos == nil {
			fmt.Println("Não foi possível recuperar seus pedidos")
			fmt.Println("")
			break
		}

		if len(pedidos) == 0 {
			fmt.Println("Não há pedidos cadastrados")
			fmt.Println("")
			break
		}

		var ids []int

		fmt.Println("Pedidos disponíveis para exclusão:")
		fmt.Println("")

		for _, pedido := range pedidos {
			ids = append(ids, pedido.Id)
		}

		fmt.Println("Digite o ID do pedido a ser excluído:")
		fmt.Println("")

		pedidoId := lerOpcaoEscolhidaListaInt(ids)

		if !service.ExcluirPedido(pedidoId) {
			fmt.Println("Não foi possível excluir o pedido.")
			fmt.Println("")
		} else {
			fmt.Println("Sucesso. Pedido excluído.")
			fmt.Println("")
		}

		fmt.Println("Deseja excluir outro pedido? (S/N)")
		resposta = lerOpcaoEscolhidaListaStr([]string{"S", "N", "s", "n"})

		fmt.Println("")
	}

	fmt.Println("Digite 1 para voltar ao menu principal")
	lerOpcaoEscolhidaMinMax(1, 1)

	criarMenu(publishCh, chavePrivada)
}

func consultarPedidos(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey) {
	fmt.Println("Como deseja consultar os pedidos?")
	fmt.Println("")
	fmt.Println("1 - Por ID")
	fmt.Println("2 - Por status/situação")
	fmt.Println("")

	opcao := lerOpcaoEscolhidaMinMax(1, 2)

	switch opcao {
	case 1:
		consultarPedidoPorId()
	case 2:
		consultarPedidosPorSituacao()
	}

	fmt.Println("Digite 1 para voltar ao menu principal")
	lerOpcaoEscolhidaMinMax(1, 1)

	criarMenu(publishCh, chavePrivada)
}

func consultarPedidoPorId() {
	fmt.Println("Digite o ID do pedido que deseja consultar:")
	fmt.Println("")

	pedidoId := lerOpcaoEscolhidaMinMax(1, math.MaxInt32)

	pedido := service.ObterPedido(pedidoId)

	if pedido == nil {
		fmt.Println("Pedido não encontrado")
		fmt.Println("")
	} else {
		fmt.Printf(
			"Pedido %d - Situação: %s\n",
			pedido.Id,
			pedido.Situacao,
		)
		fmt.Println("")
	}
}

func consultarPedidosPorSituacao() {
	fmt.Println("Digite a situação dos pedidos que deseja consultar:")
	fmt.Println("")
	fmt.Println("1 - CRIADO")
	fmt.Println("2 - ESTOQUE_DISPONIVEL")
	fmt.Println("3 - ESTOQUE_INDISPONIVEL")
	fmt.Println("4 - PAGAMENTO_APROVADO")
	fmt.Println("5 - PAGAMENTO_RECUSADO")
	fmt.Println("6 - ENVIADO")
	fmt.Println("")

	opcao := lerOpcaoEscolhidaMinMax(1, 6)

	situacoes := []string{
		"CRIADO",
		"ESTOQUE_DISPONIVEL",
		"ESTOQUE_INDISPONIVEL",
		"PAGAMENTO_APROVADO",
		"PAGAMENTO_RECUSADO",
		"ENVIADO",
	}

	situacao := situacoes[opcao-1]

	pedidos := service.ObterPedidosPorSituacao(situacao)

	if pedidos == nil {
		fmt.Println("Não foi possível consultar os pedidos")
		fmt.Println("")
		return
	}

	if len(pedidos) == 0 {
		fmt.Printf("Não existem pedidos com a situação %s\n", situacao)
		fmt.Println("")
		return
	}

	fmt.Printf("Pedidos com situação %s:\n", situacao)
	fmt.Println("")

	for _, pedido := range pedidos {
		fmt.Printf(
			"Pedido %d - Situação: %s\n",
			pedido.Id,
			pedido.Situacao,
		)
	}

	fmt.Println("")
}