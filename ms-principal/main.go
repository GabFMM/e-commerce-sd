package main

import (
	"context"
	"crypto/ed25519"
	"e-commerce-sd/ms-principal/dto"
	"e-commerce-sd/ms-principal/service"
	"e-commerce-sd/seguranca"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

const NomeExchange = "eCommerce"
const NomeServico = "principal"
const NomeFila = "principal"

// O *amqp.Channel não é thread-safe para publicar: com HTTP várias
// requisições podem chamar publicar() ao mesmo tempo.
var publishMu sync.Mutex

// Encerra o MS
func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

// Não encerra o MS
func printOnError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s\n", msg, err)
	}
}

type ItemRequest struct {
	ProdutoID  int `json:"produto_id" binding:"required,min=1"`
	Quantidade int `json:"quantidade" binding:"required,min=1"`
}

type CriarPedidoRequest struct {
	Itens []ItemRequest `json:"itens" binding:"required,min=1,dive"`
}

var situacoesValidas = []string{
	"CRIADO",
	"ESTOQUE_DISPONIVEL",
	"ESTOQUE_INDISPONIVEL",
	"PAGAMENTO_APROVADO",
	"PAGAMENTO_RECUSADO",
	"ENVIADO",
}

type PayloadEvento struct {
	Tipo        string `json:"tipo"`
	PedidoID    int    `json:"pedido_id"`
	Situacao    string `json:"situacao"`
	UrlCheckout string `json:"url_checkout,omitempty"`
}

type InteresseDTO struct {
	Email      string   `json:"email" binding:"required,email"`
	Categorias []string `json:"categorias"`
	Acao       string   `json:"acao"` // "CADASTRAR" ou "CANCELAR"
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

	msgs, err := consumeCh.Consume(
		NomeFila, // queue
		"",       // id do consumidor
		true,     // auto-ack
		true,     // exclusive
		false,    // no-local
		false,    // no-wait
		nil,      // args
	)
	failOnError(err, "[ERRO-MS-PRINCIPAL] Falha ao registrar na fila do consumidor")

	chavesPublicas, err := seguranca.CarregarTodasChavesPublicas("ms-principal/chaves/")
	failOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível carregar as chaves públicas")

	chavePrivada, err := seguranca.CarregarChavePrivada("ms-principal/chaves/principal_private.pem")
	failOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível carregar chave privada")

	// Consumer RabbitMQ em goroutine
	go consumirMensagens(msgs, publishCh, chavePrivada, chavesPublicas)

	// API HTTP
	router := montarRouter(publishCh, chavePrivada)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		log.Printf("[INFO-MS-PRINCIPAL] API escutando em %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Panicf("[ERRO-MS-PRINCIPAL] Erro no servidor HTTP: %s", err)
		}
	}()

	// Bloqueia até CTRL+C / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[INFO-MS-PRINCIPAL] Encerrando...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	printOnError(srv.Shutdown(ctx), "[ERRO-MS-PRINCIPAL] Erro no shutdown do servidor HTTP")
}

// Monta o roteador Gin com todas as rotas /api/
func montarRouter(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:4200"}, // porta do seu frontend
		AllowMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type"},
	}))

	api := r.Group("/api")

	// Método	Rota	          Substitui	            Resposta
	// GET	    /api/produtos	  visualizarProdutos	200 + lista
	// POST	    /api/pedidos	  realizarPedidos	    201 + pedido criado
	// GET	    /api/pedidos	  consultar/listar	    200 + lista (aceita ?situacao=)
	// GET	    /api/pedidos/:id  consultarPedidoPorId	200 ou 404
	// DELETE	/api/pedidos/:id  excluirPedidos	    204 ou 404
	// GET	    /api/eventos	  (novo)	            Stream SSE (fase posterior)
	{
		api.GET("/produtos", func(c *gin.Context) {
			corpoJSON, err := service.ObterProdutosDoEstoque()
			if err != nil {
				log.Printf("[ERRO-MS-PRINCIPAL] Falha ao consultar MS Estoque: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"erro": "não foi possível recuperar produtos"})
				return
			}
			c.Data(http.StatusOK, "application/json; charset=utf-8", corpoJSON)
		})

			var ids []int

			for _, idQuery := range idsQuery {
				id, err := strconv.Atoi(idQuery)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"erro": "id inválido: " + idQuery,
					})
					return
				}

				ids = append(ids, id)
			}

			produtos := service.ObterProdutosFiltros(
				nome,
				nomePresente,
				ids,
				len(idsQuery) > 0,
			)

			if produtos == nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"erro": "não foi possível recuperar produtos",
				})
				return
			}

			c.JSON(http.StatusOK, produtos)
		})

		api.GET("/pedidos", func(c *gin.Context) {
			situacao := c.Query("situacao")

			var pedidos []service.Pedido

			if situacao == "" {
				pedidos = service.ObterPedidos()
			} else {
				if !slices.Contains(situacoesValidas, situacao) {
					c.JSON(http.StatusBadRequest, gin.H{"erro": "situação inválida"})
					return
				}
				pedidos = service.ObterPedidosPorSituacao(situacao)
			}

			// nil = erro de banco (lista vazia deve vir como []Pedido{}, não nil)
			if pedidos == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"erro": "não foi possível consultar os pedidos"})
				return
			}

			c.JSON(http.StatusOK, pedidos)
		})

		api.GET("/pedidos/:id", func(c *gin.Context) {
			id, err := strconv.Atoi(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
				return
			}

			pedido := service.ObterPedido(id)
			if pedido == nil {
				c.JSON(http.StatusNotFound, gin.H{"erro": "pedido não encontrado"})
				return
			}

			c.JSON(http.StatusOK, pedido)
		})

		api.POST("/pedidos", func(c *gin.Context) {
			var req CriarPedidoRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
				return
			}

			itens := make([]service.ItemPedido, 0, len(req.Itens))
			for _, item := range req.Itens {
				itens = append(itens, service.ItemPedido{
					ProdutoId:  item.ProdutoID,
					Quantidade: item.Quantidade,
				})
			}

			id, err := service.CriarPedidoComItens(itens)
			if err != nil {
				switch {
				case errors.Is(err, service.ErrPedidoSemItens),
					errors.Is(err, service.ErrProdutoInexistente),
					errors.Is(err, service.ErrItemDuplicado):
					c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
				default:
					c.JSON(http.StatusInternalServerError, gin.H{"erro": "não foi possível criar o pedido"})
				}
				return
			}

			// Só publica depois do commit do pedido
			err = publicar(publishCh, chavePrivada, "pedido.criado", dto.PedidoDTO{Id: id})
			printOnError(err, "[ERRO-MS-PRINCIPAL] Pedido criado, mas falhou ao publicar pedido.criado")

			c.JSON(http.StatusCreated, gin.H{"id": id, "situacao": "CRIADO"})
		})

		api.DELETE("/pedidos/:id", func(c *gin.Context) {
			id, err := strconv.Atoi(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
				return
			}

			// ExcluirPedido só devolve bool, então checamos a existência antes
			// para conseguir responder 404 em vez de 500.
			if service.ObterPedido(id) == nil {
				c.JSON(http.StatusNotFound, gin.H{"erro": "pedido não encontrado"})
				return
			}

			if !service.ExcluirPedido(id) {
				c.JSON(http.StatusInternalServerError, gin.H{"erro": "não foi possível excluir o pedido"})
				return
			}

			// Avisa os outros MS
			err = publicar(publishCh, chavePrivada, "pedido.excluido", dto.PedidoDTO{Id: id})
			printOnError(err, "[ERRO-MS-PRINCIPAL] Pedido excluído, mas falhou ao publicar pedido.excluido")

			c.Status(http.StatusNoContent) // 204 não tem body
		})

		api.GET("/eventos", streamEventos)

		api.POST("/promocoes/interesse", func(c *gin.Context) {
			var req InteresseDTO
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
				return
			}

			req.Acao = "CADASTRAR"
			err := publicar(publishCh, chavePrivada, "interesse.promocao", req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"erro": "falha ao registrar interesse"})
				return
			}

			c.JSON(http.StatusCreated, gin.H{"mensagem": "Solicitação de interesse enviada"})
		})

		api.DELETE("/promocoes/interesse", func(c *gin.Context) {
			email := c.Query("email")
			if email == "" {
				c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o e-mail via ?email="})
				return
			}

			req := InteresseDTO{
				Email: email,
				Acao:  "CANCELAR",
			}

			err := publicar(publishCh, chavePrivada, "interesse.promocao", req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"erro": "falha ao cancelar interesse"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"mensagem": "Solicitação de cancelamento enviada"})
		})
	}

	return r
}

// Processa mensagens da fila e despacha por routing key.
// Ajuste o tipo de chavesPublicas para o que CarregarTodasChavesPublicas retorna.
func consumirMensagens(msgs <-chan amqp.Delivery, publishCh *amqp.Channel,
	chavePrivada ed25519.PrivateKey, chavesPublicas map[string]ed25519.PublicKey) {
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
		case "pagamento.criado":
			pagamentoCriado(payload)
		default:
			log.Printf("[ERRO-MS-PRINCIPAL] routing key inválida: %s\n", d.RoutingKey)
		}
	}
}

func notificarSSE(tipo string, pedidoID int, situacao string) {
	sseHub.broadcast(PayloadEvento{
		Tipo:     tipo,
		PedidoID: pedidoID,
		Situacao: situacao,
	})
}

func pagamentoAprovado(payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "PAGAMENTO_APROVADO")
	notificarSSE("pagamento.aprovado", pedidoDTO.Id, "PAGAMENTO_APROVADO")
}

func pagamentoRecusado(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "PAGAMENTO_RECUSADO")
	notificarSSE("pagamento.recusado", pedidoDTO.Id, "PAGAMENTO_RECUSADO")

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
	notificarSSE("pedido.enviado", pedidoDTO.Id, "ENVIADO")
}

func estoqueOk(payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "ESTOQUE_DISPONIVEL")
	notificarSSE("pedido.estoque_ok", pedidoDTO.Id, "ESTOQUE_DISPONIVEL")
}

func estoqueIndisponivel(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, payload json.RawMessage) {
	var pedidoDTO dto.PedidoDTO

	err := json.Unmarshal(payload, &pedidoDTO)
	printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
	if err != nil {
		return
	}

	service.AtualizarSituacaoPedido(pedidoDTO, "ESTOQUE_INDISPONIVEL")
	notificarSSE("estoque.indisponivel", pedidoDTO.Id, "ESTOQUE_INDISPONIVEL")

	publicar(publishCh, chavePrivada, "pedido.excluido", pedidoDTO)
}

// Cria o pacote assinado e publica no exchange. Seguro para uso concorrente.
// Retorna erro para que o handler HTTP possa responder 500 se falhar.
func publicar(publishCh *amqp.Channel, chavePrivada ed25519.PrivateKey, routingKey string, body interface{}) error {
	bodyByte, err := seguranca.CriarPacote(body, NomeServico, chavePrivada)
	if err != nil {
		log.Printf("[ERRO-MS-PRINCIPAL] Não foi possível criar o pacote para %s: %v\n", routingKey, err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	publishMu.Lock()
	defer publishMu.Unlock()

	err = publishCh.PublishWithContext(ctx,
		NomeExchange, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        bodyByte,
		},
	)
	if err != nil {
		log.Printf("[ERRO-MS-PRINCIPAL] Falha ao publicar %s: %v\n", routingKey, err)
		return err
	}

	log.Printf("[INFO-MS-PRINCIPAL] Pacote publicado para %s\n", routingKey)
	return nil
}
func pagamentoCriado(payload json.RawMessage) {
	var ev struct {
		PedidoID    int    `json:"pedido_id"`
		UrlCheckout string `json:"url_checkout"`
	}
	if err := json.Unmarshal(payload, &ev); err != nil {
		printOnError(err, "[ERRO-MS-PRINCIPAL] Não foi possível desserializar payload recebido")
		return
	}
	// A situação do pedido não muda aqui: segue ESTOQUE_DISPONIVEL
	sseHub.broadcast(PayloadEvento{
		Tipo:        "pagamento.criado",
		PedidoID:    ev.PedidoID,
		Situacao:    "ESTOQUE_DISPONIVEL",
		UrlCheckout: ev.UrlCheckout,
	})
}
