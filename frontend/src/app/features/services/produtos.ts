import { inject, Service } from '@angular/core';
import { Apollo, gql } from 'apollo-angular';

export interface ProdutoDTO {
  id: number,
  nome: string,
  categoria: string,
  quantidadeDisponivel: number
}

@Service()
export class ProdutosService {
  private apollo = inject(Apollo);

  public obterProdutos(nome?: string, ids?: number[]) {
    return this.apollo
      .query<{ produtos: ProdutoDTO[] }>({
        query: gql`
          query Produtos($ids: [Int!], $nomeLike: String){
            produtos(ids: $ids, nomeLike: $nomeLike) {
              id
              nome
              categoria
              quantidadeDisponivel
            }
          }
        `,
        variables: {
          ids: ids,
          nomeLike: nome
        },
        fetchPolicy: 'network-only'
      });
  }
}
