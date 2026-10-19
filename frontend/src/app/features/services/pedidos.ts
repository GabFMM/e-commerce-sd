import { HttpClient } from '@angular/common/http';
import { inject, Service, signal } from '@angular/core';
import { Observable } from 'rxjs';
import { ProdutoDTO, ProdutosService } from './produtos';
import { ProdutoPronto } from './carrinho';
import { ToastService, TipoToast } from './toast';

export interface PedidoResponse {
  id: number;
  situacao: string;
  // Relaciona cada produto.id com sua quantidade desejada
  produtos: Record<number, number>;
}

export interface CriarPedidoRequest {
  itens: Item[];
}

interface Item {
  produto_id: number;
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
  private http = inject(HttpClient);
  private apiUrl = 'http://localhost:8080/api/pedidos';

  protected produtosService = inject(ProdutosService);
  private toastService = inject(ToastService);

  private pedidos = signal<PedidoResponse[]>([]);

  private eventSource?: EventSource;

  constructor() {
    this.recarregarPedidos();
  }

  private obterPedidos(): Observable<PedidoResponse[]> {
    return this.http.get<PedidoResponse[]>(`${this.apiUrl}`);
  }

  public recarregarPedidos(): void {
    this.obterPedidos().subscribe({
      next: (p) => {
        this.pedidos.set(p);
      },
      error: (err) => {
        console.error(err);
      }
    });
  }

  public getPedidos(): PedidoResponse[] {
    return this.pedidos();
  }

  public criarPedido(produtosProntos: ProdutoPronto[]): Observable<void> {
    const itens: Item[] = produtosProntos.map(
      produtoPronto =>
        ({
          produto_id: produtoPronto.produtoId,
          quantidade: produtoPronto.quantidade
        })
    );

    const pedido: CriarPedidoRequest = { itens };

    return this.http.post<void>(
      `${this.apiUrl}`,
      pedido
    );
  }

  public obterProdutosPedido(pedidoId: number): Observable<ProdutoDTO[]> {
    const pedido = this.pedidos().find(
      pedido => pedido.id === pedidoId
    );

    const itens: Record<number, number> = pedido!.produtos;
    const produtosIds = Object.keys(itens).map(Number);

    return this.produtosService.obterProdutos(
      undefined,
      produtosIds
    );
  }

  public obterQuantidadeRequisitada(pedidoId: number, produtoId: number): number {
    const produtos = this.pedidos().find(pedido => pedido.id === pedidoId)!.produtos;

    for (const [id, quantidade] of Object.entries(produtos)) {
      if (Number(id) === produtoId) {
        return quantidade;
      }
    }

    // Erro: nunca ocorre
    return -1;
  }

  public removerPedido(pedidoId: number): void {
    this.http.delete(`${this.apiUrl}/${pedidoId}`).subscribe({
      next: () => {
        this.pedidos.update(pedidos => pedidos.filter(pedido => pedido.id !== pedidoId));
      },
      error: (err) => {
        console.error(err);
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

      // Atualiza a situação do pedido na lista
      this.pedidos.update(pedidos =>
        pedidos.map(p =>
          p.id === ev.pedido_id ? { ...p, situacao: ev.situacao } : p
        )
      );

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
      // Toast com link de pagamento fica até o usuário fechar
      ev.url_checkout ? null : 6000
    );
  }
}