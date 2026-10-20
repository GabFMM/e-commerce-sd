import { Component, inject } from '@angular/core';
import { Router, RouterOutlet } from '@angular/router';
import { Header } from './layout/header/header';
import { PedidosService } from './features/services/pedidos';
import { ToastContainer } from './features/services/toast-container';

@Component({
  selector: 'app-root',
  imports: [Header, RouterOutlet, ToastContainer],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  private router = inject(Router);

  constructor() {
    inject(PedidosService).conectarSSE();
  }

  protected estaNoMockPagamentos(): boolean {
    return this.router.url.startsWith('/mock-pagamento/');
  }
}