import { Component, inject, OnInit, signal } from '@angular/core';
import { ProdutoDTO, ProdutosService } from '../services/produtos';
import { CarrinhoService } from '../services/carrinho';
import { FormsModule } from '@angular/forms';

@Component({
  imports: [FormsModule],
  selector: 'app-catalogo',
  styleUrl: './catalogo.css',
  templateUrl: './catalogo.html',
})
export class Catalogo implements OnInit {
  protected carrinhoSerice = inject(CarrinhoService);
  protected produtosService = inject(ProdutosService);

  protected produtos = signal<ProdutoDTO[]>([]);
  protected nomeProduto: string = "";

  ngOnInit(): void {
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
