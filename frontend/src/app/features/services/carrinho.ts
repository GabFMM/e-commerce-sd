import { Service, signal } from '@angular/core';
import { Produto } from './produtos';

interface ProdutoPronto {
    produtoId: number,
    // Quantidade desejada pelo usuário
    quantidade: number
}

@Service()
export class CarrinhoService {
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

    public limpar(): void {
        this.produtos.set([]);
    }
}
