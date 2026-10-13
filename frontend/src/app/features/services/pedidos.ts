import { HttpClient } from '@angular/common/http';
import { inject, Service, signal } from '@angular/core';
import { Observable } from 'rxjs';
import { ProdutoDTO, ProdutosService } from './produtos';
import { ProdutoPronto } from './carrinho';

export interface PedidoDTO {
  id: number;
  situacao: string;
  // Relaciona cada produto.id com sua quantidade desejada
  itens: Record<number, number>;
}

export interface CriarPedidoDTO {
  // Relaciona cada produto.id com sua quantidade desejada
  itens: Record<number, number>;
}

@Service()
export class PedidosService {
  private http = inject(HttpClient);
  private apiUrl = 'http://localhost:8080/pedidos';

  protected produtosService = inject(ProdutosService);

  private pedidos = signal<PedidoDTO[]>([]);

  constructor() {
    this.obterPedidos().subscribe({
      next: (p) => {
        this.pedidos.set(p);
      },
      error: (err) => {
        console.error(err);
      }
    });
  }

  private obterPedidos(): Observable<PedidoDTO[]> {
    return this.http.get<PedidoDTO[]>(`${this.apiUrl}`);
  }

  public getPedidos(): PedidoDTO[] {
    return this.pedidos();
  }

  public criarPedido(produtosProntos: ProdutoPronto[]): Observable<void> {
    const pedido: CriarPedidoDTO = {
      itens: produtosProntos.reduce((itens, produtoPronto) => {
        itens[produtoPronto.produtoId] = produtoPronto.quantidade;
        return itens;
      }, {} as Record<number, number>)
    };

    return this.http.post<void>(
      `${this.apiUrl}`,
      pedido
    );
  }

  public obterProdutosPedido(pedidoId: number): ProdutoDTO[] {
    let produtosPedido: ProdutoDTO[] = [];

    const itens: Record<number, number> = this.pedidos().find(pedido => pedido.id === pedidoId)!.itens;

    const produtosIds: number[] = [];
    for(const id of Object.keys(itens)) {
      produtosIds.push(Number(id));
    }

    this.produtosService.obterProdutos(undefined, produtosIds).subscribe({
      next: (produtos) => {
        produtosPedido = produtos;
      },
      error: (err) => {
        console.error(err);
      }
    });

    return produtosPedido;
  }

  public removerPedido(pedidoId: number): void {
    this.http.delete(`${this.apiUrl}/${pedidoId}`).subscribe({
      next: () => {
        this.pedidos.update(pedidos => pedidos.filter(pedido => pedido.id !== pedidoId));
      },
      error: (err) => {
        console.error(err);
      }
    })
  }
}
