import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { InteresseRequest, NotificacoesService } from '../services/notificacoes';

@Component({
  imports: [FormsModule],
  selector: 'app-notificacoes',
  styleUrl: './notificacoes.css',
  templateUrl: './notificacoes.html',
})
export class Notificacoes {
  private notificacoesService = inject(NotificacoesService);

  protected interessesAtualizados = signal(false);

  protected categoriaA = false;
  protected categoriaB = false;
  protected categoriaC = false;
  protected email: string = "";

  protected haPeloMenosUmaCategoriaSelecionada(): boolean {
    return this.categoriaA || this.categoriaB || this.categoriaC;
  }

  protected atualizarInteresses(): void {
    const interesseRequest: InteresseRequest = {
      email: this.email,
      categorias: [
        ...(this.categoriaA ? ['A'] : []),
        ...(this.categoriaB ? ['B'] : []),
        ...(this.categoriaC ? ['C'] : [])
      ],
      acao: "CADASTRAR"
    };

    this.notificacoesService.atualizarInteresses(interesseRequest).subscribe({
      next: () => {
        this.interessesAtualizados.set(true);
      },
      error: (err) => {
        console.error(err);
      }
    })
  }
}
