import { inject, Service, signal } from '@angular/core';
import { Produto } from './produtos';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

interface ProdutoPronto {
  produtoId: number,
  // Quantidade desejada pelo usuário
  quantidade: number
}

interface Pedido {
  // Relaciona cada produto.id com sua quantidade desejada
  itens: Record<number, number>;
}

@Service()
export class CarrinhoService {
  private http = inject(HttpClient);
  private apiUrl = 'http://localhost:8080';

  // Produtos com quantidade desejada
  private produtosProntos: ProdutoPronto[] = [];

  // Produtos marcados como desejados,
  // mas não foi determinada a quantidade desejada
  protected produtos = signal<Produto[]>([]);

  public getProdutos(): Produto[] {
    return this.produtos();
  }

  public temProdutoPronto(): boolean {
    return Boolean(this.produtosProntos.length);
  }

  public produtoEstaNoCarrinho(produto: Produto): boolean {
    return this.produtos().some((p) => p.id === produto.id);
  }

  public adicionarProduto(produto: Produto): void {
    // Não inclui duas vezes o mesmo produto
    if(this.produtoEstaNoCarrinho(produto))
      return;

    this.produtos.update((p) => [...p, produto]);
  }

  public adicionarProdutoPronto(produtoId: number, quantidadeDesejada: number): void {
    this.produtosProntos.push({
      produtoId: produtoId,
      quantidade: quantidadeDesejada
    });
  }

  public removerProduto(produtoId: number): void {
    this.produtos.update(itens => itens.filter(item => item.id !== produtoId));
  }

  // Remove os produtos do carrinho
  // que possuem uma quantidade desejada
  public limpar(): void {
    this.produtos.update(
      produtos => produtos.filter(
        produto => !this.produtosProntos.some(
          produtoPronto => produtoPronto.produtoId === produto.id
        )
      )
    );

    this.produtosProntos = [];
  }

  public criarPedido(): Observable<void> {
    const pedido: Pedido = {
      itens: this.produtosProntos.reduce((itens, produtoPronto) => {
        itens[produtoPronto.produtoId] = produtoPronto.quantidade;
        return itens;
      }, {} as Record<number, number>)
    };

    return this.http.post<void>(
      `${this.apiUrl}/pedidos`,
      pedido
    );
  }
}
