import { Component, inject } from '@angular/core';
import { ToastService } from '../services/toast';

@Component({
  selector: 'app-toast-container',
  template: `
    <div class="container">
      @for (t of servico.toasts(); track t.id) {
        <div class="toast" [class]="t.tipo" role="alert">
          <div class="conteudo">
            <strong>{{ t.titulo }}</strong>
            <span>{{ t.mensagem }}</span>
            @if (t.link) {
              <a [href]="t.link.url" target="_blank" rel="noopener">{{ t.link.texto }}</a>
            }
          </div>
          <button type="button" aria-label="Fechar" (click)="servico.fechar(t.id)">×</button>
        </div>
      }
    </div>
  `,
  styles: `
    .container {
      position: fixed;
      top: 16px;
      right: 16px;
      z-index: 9999;
      display: flex;
      flex-direction: column;
      gap: 10px;
      width: min(360px, calc(100vw - 32px));
      pointer-events: none; /* não bloqueia cliques na página */
    }
    .toast {
      pointer-events: auto;
      display: flex;
      justify-content: space-between;
      gap: 12px;
      padding: 12px 14px;
      border-radius: 8px;
      border-left: 5px solid;
      color: #fff;
      box-shadow: 0 4px 14px rgba(0, 0, 0, 0.25);
      animation: entrar 0.25s ease-out;
    }
    .conteudo { display: flex; flex-direction: column; gap: 2px; }
    .conteudo a { color: #fff; text-decoration: underline; margin-top: 4px; }
    button {
      background: none; border: none; color: inherit;
      font-size: 20px; line-height: 1; cursor: pointer; opacity: 0.8;
    }
    button:hover { opacity: 1; }

    .sucesso { background: #2e7d32; border-color: #1b5e20; }
    .erro    { background: #c62828; border-color: #8e0000; }
    .aviso   { background: #ef6c00; border-color: #b53d00; }
    .info    { background: #1565c0; border-color: #003c8f; }

    @keyframes entrar {
      from { transform: translateX(110%); opacity: 0; }
      to   { transform: translateX(0);    opacity: 1; }
    }
  `,
})
export class ToastContainer {
  protected servico = inject(ToastService);
}