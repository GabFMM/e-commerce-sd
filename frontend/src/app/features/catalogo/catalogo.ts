import { Component, inject, OnInit, signal } from '@angular/core';
import { ProdutoDTO, ProdutosService } from '../services/produtos';
import { CarrinhoService } from '../services/carrinho';
import { FormsModule } from '@angular/forms';
import { ToastService } from '../services/toast';

@Component({
  imports: [FormsModule],
  selector: 'app-catalogo',
  styleUrl: './catalogo.css',
  templateUrl: './catalogo.html',
})
export class Catalogo implements OnInit {
  protected carrinhoSerice = inject(CarrinhoService);
  protected produtosService = inject(ProdutosService);
  protected toastService = inject(ToastService);

  protected produtos = signal<ProdutoDTO[]>([]);
  protected nomeProduto: string = "";

  ngOnInit(): void {
    this.produtosService.obterProdutos().subscribe({
      next: ({ data, error }) => {
        if (error) {
          console.error('Erro GraphQL ao buscar produtos:', error);
          return;
        }

        if (data === undefined) {
          console.error('A API não retornou dados dos produtos.');
          return;
        }

        this.produtos.set(data.produtos);
      },
      error: (err) => {
        console.error('Erro de comunicação ao buscar produtos:', err);
      }
    });
  }

  protected pesquisarProdutos(): void {
    this.produtosService.obterProdutos(this.nomeProduto).subscribe({
      next: ({ data, error }) => {
        if (error) {
          console.error('Erro GraphQL ao buscar produtos:', error);
          return;
        }

        if (data === undefined) {
          console.error('A API não retornou dados dos produtos.');
          return;
        }

        this.produtos.set(data.produtos);
      },
      error: (err) => {
        console.error('Erro de comunicação ao buscar produtos:', err);
      }
    })
  }

  protected AdicionarProdutoCarrinho(produto: ProdutoDTO): void {
    this.carrinhoSerice.adicionarProduto(produto);
    this.toastService.exibir({
        tipo: 'info',
        titulo: 'Informação',
        mensagem: 'Produto adicionado no carrinho'
      },
      3000);
  }
}
