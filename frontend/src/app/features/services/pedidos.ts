import { inject, Service, signal } from '@angular/core';
import { ProdutoPronto } from './carrinho';
import { ToastService, TipoToast } from './toast';
import { Apollo, gql } from 'apollo-angular';
import { ProdutosService } from './produtos';

export interface PedidoResponse {
  id: number;
  situacao: string;
  produtos: Item[];
}

export interface CriarPedidoRequest {
  itens: Item[];
}

interface Item {
  produtoId: number;
  quantidade: number;
}

// Formato enviado pelo backend (PayloadEvento no main.go)
export interface EventoPedido {
  tipo: string;
  pedido_id: number;
  situacao: string;
  url_checkout?: string;
}

const CONFIG_EVENTOS: Record<
  string,
  { tipo: TipoToast; titulo: string; texto: (id: number) => string }
> = {
  'pedido.estoque_ok':    { tipo: 'info',    titulo: 'Estoque confirmado',   texto: id => `Pedido #${id}: produtos reservados.` },
  'estoque.indisponivel': { tipo: 'erro',    titulo: 'Estoque indisponível', texto: id => `Pedido #${id} cancelado por falta de estoque.` },
  'pagamento.criado':     { tipo: 'aviso',   titulo: 'Aguardando pagamento', texto: id => `Pedido #${id}: finalize o pagamento.` },
  'pagamento.aprovado':   { tipo: 'sucesso', titulo: 'Pagamento aprovado',   texto: id => `Pedido #${id} foi pago.` },
  'pagamento.recusado':   { tipo: 'erro',    titulo: 'Pagamento recusado',   texto: id => `Pedido #${id} teve o pagamento recusado.` },
  'pedido.enviado':       { tipo: 'sucesso', titulo: 'Pedido enviado',       texto: id => `Pedido #${id} saiu para entrega.` },
};

@Service()
export class PedidosService {
  private apollo = inject(Apollo);

  protected produtosService = inject(ProdutosService);
  private toastService = inject(ToastService);

  private pedidos = signal<PedidoResponse[]>([]);

  private eventSource?: EventSource;

  constructor() {
    this.recarregarPedidos();
  }

  private obterPedidos() {
    return this.apollo
      .query<{ pedidos: PedidoResponse[] }>({
        query: gql`
          {
            pedidos {
              id
              situacao
              produtos {
                produtoId
                quantidade
              }
            }
          }
        `,
        fetchPolicy: 'network-only'
      });
  }

  public recarregarPedidos(): void {
    this.obterPedidos().subscribe({
      next: ({ data, error }) => {
        if (error) {
          console.error('Erro GraphQL ao buscar pedidos:', error);
          return;
        }

        if (data === undefined) {
          console.error('A API não retornou dados dos pedidos.');
          return;
        }

        this.pedidos.set(data.pedidos);
      },
      error: (error) => {
        console.error(error);
      }
    });
  }

  public getPedidos(): PedidoResponse[] {
    return this.pedidos();
  }

  public criarPedido(produtosProntos: ProdutoPronto[]) {
    const itens: Item[] = produtosProntos.map(
      produtoPronto =>
        ({
          produtoId: produtoPronto.produtoId,
          quantidade: produtoPronto.quantidade
        })
    );

    const request: CriarPedidoRequest = { itens };

    return this.apollo
      .mutate<{ criarPedido: boolean }>({
        mutation: gql`
          mutation CriarPedido($request: CriarPedidoRequest!){
            criarPedido(request: $request)
          }
        `,
        variables: {
          request: request
        }
      });
  }

  public obterProdutosPedido(pedidoId: number) {
    const pedido = this.pedidos().find(
      pedido => pedido.id === pedidoId
    );

    const produtosIds = pedido!.produtos.map(item => item.produtoId);

    return this.produtosService.obterProdutos(
      undefined,
      produtosIds
    );
  }

  public obterQuantidadeRequisitada(pedidoId: number, produtoId: number): number {
    const pedido = this.pedidos().find(pedido => pedido.id === pedidoId);

    const item = pedido?.produtos.find(p => p.produtoId === produtoId);
    return item?.quantidade ?? -1;
  }

  public removerPedido(pedidoId: number): void {
    this.apollo
      .mutate<{ removerPedido: boolean }>({
        mutation: gql`
          mutation RemoverPedido($request: RemoverPedidoRequest!){
            removerPedido(request: $request)
          }
        `,
        variables: {
          request: { id: pedidoId }
        }
      })
      .subscribe({
        next: ({ data, error }) => {
          if (data === undefined) {
            console.error('A API não retornou dados.');
            return;
          }

          if (!data.removerPedido) {
            console.error('Não foi possível remover o pedido.');
            return;
          }

          this.pedidos.update(pedidos => pedidos.filter(pedido => pedido.id !== pedidoId));
        },
        error: (error) => {
          console.error(error);
        }
      });
  }

  public conectarSSE(): void {
    // Evita criar conexões duplicadas.
    if (this.eventSource) {
      return;
    }

    this.eventSource = new EventSource('http://localhost:8080/api/eventos');

    // Ao (re)conectar, sincroniza o estado: eventos enviados sem cliente conectado são perdidos
    this.eventSource.onopen = () => {
      this.recarregarPedidos();
    };

    // O backend usa o nome de evento "message" (c.SSEvent("message", ev))
    this.eventSource.addEventListener('message', (e: MessageEvent) => {
      const ev: EventoPedido = JSON.parse(e.data);

      // Significa que um pedido foi criado
      if (ev.tipo === "pedido.estoque_ok") {
        this.recarregarPedidos();
      }
      // Atualiza a situação do pedido existente na lista em memória
      else {
        this.pedidos.update(pedidos =>
          pedidos.map(p =>
            p.id === ev.pedido_id ? { ...p, situacao: ev.situacao } : p
          )
        );
      }
      this.notificar(ev);
    });

    this.eventSource.onerror = erro => {
      console.error('Erro na conexão SSE:', erro);
      // EventSource tenta reconectar automaticamente.
    };
  }

  public desconectarSSE(): void {
    this.eventSource?.close();
    this.eventSource = undefined;
  }

  private notificar(ev: EventoPedido): void {
    const config = CONFIG_EVENTOS[ev.tipo];
    if (!config) return;

    this.toastService.exibir(
      {
        tipo: config.tipo,
        titulo: config.titulo,
        mensagem: config.texto(ev.pedido_id),
        link: ev.url_checkout
          ? { texto: 'Ir para o pagamento', url: ev.url_checkout }
          : undefined,
      },
      // Toast com link de pagamento tem uma duração maior de tela
      ev.url_checkout ? 10000 : 6000
    );
  }
}