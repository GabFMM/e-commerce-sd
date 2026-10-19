import { Routes } from '@angular/router';
import { Catalogo } from './features/catalogo/catalogo';
import { Carrinho } from './features/carrinho/carrinho';
import { Pedidos } from './features/pedidos/pedidos';
import { Notificacoes } from './features/notificacoes/notificacoes';

export const routes: Routes = [
  {
    path: '',
    component: Catalogo
  },
  {
    path: 'carrinho',
    component: Carrinho
  },
  {
    path: 'pedidos',
    component: Pedidos
  },
  {
    path: 'notificacoes',
    component: Notificacoes
  },
  {
    path: '**',
    redirectTo: '',
    pathMatch: 'full'
  }
];
