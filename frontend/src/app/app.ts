import { Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
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
  constructor() {
    inject(PedidosService).conectarSSE();
  }
}