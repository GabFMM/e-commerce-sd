import { Component, inject, signal } from '@angular/core';
import { ProdutoDTO, ProdutosService } from '../services/produtos';
import { CarrinhoService } from '../services/carrinho';

@Component({
  imports: [],
  selector: 'app-catalogo',
  styleUrl: './catalogo.css',
  templateUrl: './catalogo.html',
})
export class Catalogo {
  protected carrinhoSerice = inject(CarrinhoService);
  protected produtosService = inject(ProdutosService);

  protected produtos = signal<ProdutoDTO[]>([]);
  protected nomeProduto: string = "";

  constructor() {
    this.produtosService.obterProdutos().subscribe({
      next: (produtos) => {
        this.produtos.set(produtos);
      },
      error: (err) => {
        console.error(err);
      }
    })
  }

  protected pesquisarProdutos(): void {
    this.produtosService.obterProdutos(this.nomeProduto).subscribe({
      next: (produtos) => {
        this.produtos.set(produtos);
      },
      error: (err) => {
        console.error(err);
      }
    })
  }

  protected AdicionarProdutoCarrinho(produto: ProdutoDTO): void {
    this.carrinhoSerice.adicionarProduto(produto);
  }
}
