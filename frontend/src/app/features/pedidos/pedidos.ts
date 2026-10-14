import { Component, inject, OnDestroy, OnInit, signal } from '@angular/core';
import { PedidosService } from '../services/pedidos';
import { RouterLink } from '@angular/router';
import { ProdutoDTO, ProdutosService } from '../services/produtos';

@Component({
  imports: [RouterLink],
  selector: 'app-pedidos',
  styleUrl: './pedidos.css',
  templateUrl: './pedidos.html',
})
export class Pedidos implements OnInit, OnDestroy {
  protected produtosService = inject(ProdutosService);
  protected pedidosService = inject(PedidosService);

  protected mostrarProdutosPedidoSignal = signal(false);
  protected pedidoId: number | null = null;
  protected produtosPedido = signal<ProdutoDTO[]>([]);

  ngOnInit(): void {
    this.pedidosService.conectarSSE();
  }

  ngOnDestroy(): void {
    this.pedidosService.desconectarSSE();
  }

  protected mostrarProdutosPedido(pedidoId: number): void {
    this.pedidoId = pedidoId;

    this.pedidosService.obterProdutosPedido(pedidoId).subscribe({
      next: (produtos) => {
        this.produtosPedido.set(produtos);
        this.mostrarProdutosPedidoSignal.set(true);
      },
      error: (err) => {
        console.error('Erro ao obter produtos do pedido:', err);
      }
    });
  }

  protected removerPedido(pedidoId: number): void {
    this.pedidosService.removerPedido(pedidoId);
  }

  protected fecharDialog(): void {
    this.pedidoId = null;
    this.mostrarProdutosPedidoSignal.set(false);
  }

  protected obterQuantidadeRequisitada(produtoId: number): number {
    return this.pedidosService.obterQuantidadeRequisitada(this.pedidoId!, produtoId);
  }
}
