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

  public getQuantidadeDisponivelProduto(produtoId: number) {
    return this.produtos().find(produto => produto.id === produtoId)!.quantidadeDisponivel;
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

  public atualizarProdutoPronto(produtoId: number, quantidadeDesejada: number): void {
    if(this.ehQuantidadeDesejadaValida(produtoId, quantidadeDesejada)) {
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
    // Por mais que nao seja uma quantidade desejada valida, 
    // pode haver um produto pronto criado com uma quantidade anterior valida,
    // logo é preciso remover o produto pronto 
    // para impedir a criação de pedidos que não corresponde com os inputs do usuario
    else {
      this.produtosProntos = this.produtosProntos.filter(
        produtoPronto => produtoPronto.produtoId !== produtoId
      );
    }
  }

  public ehQuantidadeDesejadaValida(produtoId: number, quantidadeDesejada: number): boolean {
    return quantidadeDesejada > 0 && quantidadeDesejada <= this.getQuantidadeDisponivelProduto(produtoId);
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
