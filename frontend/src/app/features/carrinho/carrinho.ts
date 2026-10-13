import { Component, inject } from '@angular/core';
import { CarrinhoService } from '../services/carrinho'
import { ProdutosService } from '../services/produtos';
import { RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';

@Component({
  imports: [RouterLink, FormsModule],
  selector: 'app-carrinho',
  styleUrl: './carrinho.css',
  templateUrl: './carrinho.html',
})
export class Carrinho {
  protected carrinhoService = inject(CarrinhoService);
  protected produtosService = inject(ProdutosService);

  // Relaciona o pedido.id com a quantidade desejada correspondente
  protected quantidadesDesejadas: Record<number, number> = {};

  protected desabilitarBotaoCriarPedido(): boolean {
    return !this.carrinhoService.temProdutoPronto();
  }

  protected criarPedido(): void {
    for (const produto of this.carrinhoService.getProdutos()) {
        const quantidade = this.quantidadesDesejadas[produto.id];

        if (quantidade > 0) {
          this.carrinhoService.adicionarProdutoPronto(produto.id, quantidade);
        }
    }
  }
}
