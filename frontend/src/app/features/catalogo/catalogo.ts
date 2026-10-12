import { Component, inject } from '@angular/core';
import { Produto, Produtos } from '../services/produtos';

@Component({
  imports: [],
  selector: 'app-catalogo',
  styleUrl: './catalogo.css',
  templateUrl: './catalogo.html',
})
export class Catalogo {
  private produtosService = inject(Produtos);

  protected produtos: Produto[] | null = null;
  protected nomeProduto: string = "";

  constructor() {
    this.produtosService.obterProdutos().subscribe({
      next: (produtos) => {
        this.produtos = produtos;
      },
      error: (err) => {
        console.error(err);
      }
    })
  }

  protected pesquisarProdutos(): void {
    this.produtosService.obterProdutos(this.nomeProduto).subscribe({
      next: (produtos) => {
        this.produtos = produtos;
      },
      error: (err) => {
        console.error(err);
      }
    })
  }
}
