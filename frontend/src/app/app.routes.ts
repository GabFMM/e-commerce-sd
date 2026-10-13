import { Routes } from '@angular/router';
import { Catalogo } from './features/catalogo/catalogo';
import { Carrinho } from './features/carrinho/carrinho';
import { Pedidos } from './features/pedidos/pedidos';

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
    path: '**',
    redirectTo: '',
    pathMatch: 'full'
  }
];
