import { Component, inject, signal } from '@angular/core';
import { PedidosService } from '../services/pedidos';
import { RouterLink } from '@angular/router';
import { ProdutoDTO, ProdutosService } from '../services/produtos';

@Component({
  imports: [RouterLink],
  selector: 'app-pedidos',
  styleUrl: './pedidos.css',
  templateUrl: './pedidos.html',
})
export class Pedidos {
  protected produtosService = inject(ProdutosService);
  protected pedidosService = inject(PedidosService);

  protected mostrarProdutosPedidoSignal = signal(false);
  protected pedidoId: number | null = null;
  protected produtosPedido = signal<ProdutoDTO[]>([]);

  // A conexão SSE agora é aberta uma única vez no App (app.ts),
  // por isso este componente não conecta nem desconecta mais.

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