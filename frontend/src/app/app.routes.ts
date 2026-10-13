import { Routes } from '@angular/router';
import { Catalogo } from './features/catalogo/catalogo';
import { Carrinho } from './features/carrinho/carrinho';

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
        path: '**',
        redirectTo: '',
        pathMatch: 'full'
    }
];
