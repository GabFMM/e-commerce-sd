import { Routes } from '@angular/router';
import { Catalogo } from './features/catalogo/catalogo';
import { Carrinho } from './features/carrinho/carrinho';
import { Pedidos } from './features/pedidos/pedidos';
import { Notificacoes } from './features/notificacoes/notificacoes';
import { MockPagamento } from './features/mock-pagamento/mock-pagamento';
 

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
    path: 'mock-pagamento/:pedidoId',
    component: MockPagamento
  },
  {
    path: '**',
    redirectTo: '',
    pathMatch: 'full'
  }
];
