import { HttpClient } from '@angular/common/http';
import { Component, inject, signal } from '@angular/core';
import { ActivatedRoute } from '@angular/router';

type Estado = 'aguardando' | 'enviando' | 'aprovado' | 'recusado' | 'erro';

@Component({
  selector: 'app-mock-pagamento',
  styleUrl: './mock-pagamento.css',
  templateUrl: './mock-pagamento.html',
})
export class MockPagamento {
  private http = inject(HttpClient);

  // Mock de pagamento (Go). Ele repassa o resultado ao webhook do MS Pagamento.
  private mockUrl = 'http://localhost:8082/checkout';

  protected pedidoId = Number(inject(ActivatedRoute).snapshot.paramMap.get('pedidoId'));
  protected estado = signal<Estado>('aguardando');
  protected mensagemErro = signal('');

  protected responder(status: 'APROVADO' | 'RECUSADO'): void {
    this.estado.set('enviando');

    this.http.post(`${this.mockUrl}/${this.pedidoId}`, { status }).subscribe({
      next: () => {
        this.estado.set(status === 'APROVADO' ? 'aprovado' : 'recusado');
      },
      error: (err) => {
        console.error(err);
        // 404 = cobrança já respondida (ou mock reiniciado)
        this.mensagemErro.set(err.error?.erro ?? 'Não foi possível contactar o mock de pagamento.');
        this.estado.set('erro');
      }
    });
  }
}