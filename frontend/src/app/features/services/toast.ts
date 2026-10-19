import { Service, signal } from '@angular/core';

export type TipoToast = 'sucesso' | 'erro' | 'aviso' | 'info';

export interface Toast {
  id: number;
  tipo: TipoToast;
  titulo: string;
  mensagem: string;
  link?: { texto: string; url: string };
}

@Service()
export class ToastService {
  private proximoId = 0;
  readonly toasts = signal<Toast[]>([]);

  // duracaoMs = null -> o toast fica até o usuário fechar
  exibir(dados: Omit<Toast, 'id'>, duracaoMs: number | null = 6000): void {
    const id = ++this.proximoId;
    this.toasts.update(lista => [...lista, { id, ...dados }]);

    if (duracaoMs !== null) {
      setTimeout(() => this.fechar(id), duracaoMs);
    }
  }

  fechar(id: number): void {
    this.toasts.update(lista => lista.filter(t => t.id !== id));
  }
}