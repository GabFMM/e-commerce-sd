import { Component, inject, OnInit } from '@angular/core';
import { CarrinhoService } from '../services/carrinho'
import { ProdutosService } from '../services/produtos';
import { RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { PedidosService } from '../services/pedidos';

@Component({
  imports: [RouterLink, FormsModule],
  selector: 'app-carrinho',
  styleUrl: './carrinho.css',
  templateUrl: './carrinho.html',
})
export class Carrinho implements OnInit {
  protected pedidosService = inject(PedidosService);
  protected carrinhoService = inject(CarrinhoService);
  protected produtosService = inject(ProdutosService);

  // Relaciona o pedido.id com a quantidade desejada correspondente
  protected quantidadesDesejadas: Record<number, number> = {};

  ngOnInit(): void {
    this.carrinhoService.getProdutosProntos().forEach(
      produtoPronto => this.quantidadesDesejadas[produtoPronto.produtoId] = produtoPronto.quantidade
    );
  }

  protected removerProduto(produtoId: number): void {
    delete this.quantidadesDesejadas[produtoId];
    this.carrinhoService.removerProduto(produtoId);
  }

  protected desabilitarBotaoCriarPedido(): boolean {
    return !this.carrinhoService.temProdutoPronto();
  }

  protected criarPedido(): void {
    this.pedidosService.criarPedido(this.carrinhoService.getProdutosProntos()).subscribe({
      next: () => {
        this.pedidosService.recarregarPedidos();
        this.carrinhoService.limpar();
      },
      error: (err) => {
        console.error(err);
      }
    })
  }
}
