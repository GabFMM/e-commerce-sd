import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Service } from '@angular/core';
import { Observable } from 'rxjs';

export interface ProdutoDTO {
  id: number,
  nome: string,
  categoria: string,
  quantidade: number,
  reservados: number
}

@Service()
export class ProdutosService {
  private http = inject(HttpClient);
  private apiUrl = 'http://localhost:8080/produtos';

  public obterProdutos(nome?: string, ids?: number[]): Observable<ProdutoDTO[]> {
    let params = new HttpParams();

    if(nome && nome.trim()) {
        params = params.set('nome_like', nome.trim());
    }

    if (ids && ids.length > 0) {
      ids.forEach(id => {
        params = params.append('id', id.toString());
      });
    }

    return this.http.get<ProdutoDTO[]>(`${this.apiUrl}`, { params });
  }

  public estaDisponivel(produto: ProdutoDTO): boolean {
    return Boolean(this.getQuantidadeDisponivel(produto));
  }

  public getQuantidadeDisponivel(produto: ProdutoDTO): number {
    return produto.quantidade - produto.reservados;
  }
}
