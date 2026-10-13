import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Service } from '@angular/core';
import { Observable } from 'rxjs';

export interface Produto {
    id: number,
    nome: string,
    categoria: string,
    quantidade: number,
    reservados: number
}

@Service()
export class ProdutosService {
    private http = inject(HttpClient);
    private apiUrl = 'http://localhost:8080';

    public obterProdutos(nomeProduto?: string): Observable<Produto[]> {
        let params = new HttpParams();

        // Só adiciona o parâmetro 'nome' se ele realmente existir e não for uma string vazia
        if (nomeProduto && nomeProduto.trim()) {
            params = params.set('nome_like', nomeProduto.trim());
        }

        return this.http.get<Produto[]>(`${this.apiUrl}/produtos`, { params });
    }

    public estaDisponivel(produto: Produto): boolean {
        return Boolean(this.getQuantidadeDisponivel(produto));
    }

    public getQuantidadeDisponivel(produto: Produto): number {
        return produto.quantidade - produto.reservados
    }
}
