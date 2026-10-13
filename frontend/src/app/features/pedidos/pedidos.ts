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
    this.produtosPedido.set(this.pedidosService.obterProdutosPedido(pedidoId));
    this.mostrarProdutosPedidoSignal.set(true);
  }

  protected removerPedido(pedidoId: number): void {
    this.pedidosService.removerPedido(pedidoId);
  }

  protected fecharDialog(): void {
    this.pedidoId = null;
    this.mostrarProdutosPedidoSignal.set(false);
  }
}
