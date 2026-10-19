import { HttpClient } from '@angular/common/http';
import { inject, Service } from '@angular/core';
import { Observable } from 'rxjs';

export interface InteresseRequest {
    email: string,
    categorias: string[],
    acao: string
}

@Service()
export class NotificacoesService {
  private http = inject(HttpClient);
  private apiUrl = 'http://localhost:8080/api/promocoes/interesse';

  public atualizarInteresses(interesseRequest: InteresseRequest): Observable<void> {
    return this.http.post<void>(this.apiUrl, interesseRequest);
  }
}
