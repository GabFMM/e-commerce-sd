import { Service, signal } from '@angular/core';
import { ProdutoDTO } from './produtos';

export interface ProdutoPronto {
  produtoId: number,
  // Quantidade desejada pelo usuário
  quantidade: number
}

@Service()
export class CarrinhoService {
  // Produtos com quantidade desejada
  private produtosProntos: ProdutoPronto[] = [];

  // Produtos marcados como desejados,
  // mas não foi determinada a quantidade desejada
  private produtos = signal<ProdutoDTO[]>([]);

  public getProdutos(): ProdutoDTO[] {
    return this.produtos();
  }

  public getProdutosProntos(): ProdutoPronto[] {
    return this.produtosProntos;
  }

  public temProdutoPronto(): boolean {
    return Boolean(this.produtosProntos.length);
  }

  public produtoEstaNoCarrinho(produto: ProdutoDTO): boolean {
    return this.produtos().some((p) => p.id === produto.id);
  }

  public adicionarProduto(produto: ProdutoDTO): void {
    // Não inclui duas vezes o mesmo produto
    if(this.produtoEstaNoCarrinho(produto))
      return;

    this.produtos.update((p) => [...p, produto]);
  }

  public adicionarProdutoPronto(produtoId: number, quantidadeDesejada: number): void {
    // Se não existe o produto pronto, cria-se um novo
    if(!this.produtosProntos.some(produtoPronto => produtoPronto.produtoId === produtoId)) {
      this.produtosProntos.push({
        produtoId: produtoId,
        quantidade: quantidadeDesejada
      });
    }
    // Senão atualiza a quantidade para impedir dois produtos prontos com o mesmo ID
    else {
      const i = this.produtosProntos.findIndex(produtoPronto => produtoPronto.produtoId === produtoId);
      this.produtosProntos[i].quantidade = quantidadeDesejada;
    }
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
}
