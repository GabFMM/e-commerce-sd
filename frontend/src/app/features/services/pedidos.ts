import { HttpClient } from '@angular/common/http';
import { inject, OnInit, Service, signal } from '@angular/core';
import { Observable } from 'rxjs';
import { ProdutoDTO, ProdutosService } from './produtos';
import { ProdutoPronto } from './carrinho';

export interface PedidoResponse {
  id: number;
  situacao: string;
  // Relaciona cada produto.id com sua quantidade desejada
  produtos: Record<number, number>;
}

export interface CriarPedidoRequest {
  // Relaciona cada produto.id com sua quantidade desejada
  itens: Item[];
}

interface Item {
  produto_id: number,
  quantidade: number,
}

export interface AtualizarPedidoResponse {
  id: number;
  situacao: string;
}

@Service()
export class PedidosService {
  private http = inject(HttpClient);
  private apiUrl = 'http://localhost:8080/api/pedidos';

  protected produtosService = inject(ProdutosService);

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

    const pedido: CriarPedidoRequest = {itens};

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

    for(const [id, quantidade] of Object.entries(produtos)) {
      if(Number(id) === produtoId) {
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

      this.eventSource = new EventSource(`${this.apiUrl}/eventos`);

      this.eventSource.addEventListener(
        'atualizar-pedido',
        (evento: MessageEvent) => {
          const atualizacao: AtualizarPedidoResponse = JSON.parse(evento.data);

          this.pedidos.update(pedidos =>
            pedidos.map(pedido =>
              pedido.id === atualizacao.id
                ? {
                  ...pedido,
                  situacao: atualizacao.situacao
                }
                : pedido
            )
          );
        }
      );

      this.eventSource.onerror = erro => {
          console.error('Erro na conexão SSE:', erro);
          // EventSource tenta reconectar automaticamente.
      };
    }

    public desconectarSSE(): void {
        this.eventSource?.close();
        this.eventSource = undefined;
    }
}
