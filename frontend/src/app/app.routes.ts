import { Routes } from '@angular/router';
import { Catalogo } from './features/catalogo/catalogo';

export const routes: Routes = [
    {
        path: '',
        component: Catalogo
    },
    {
        path: '**',
        redirectTo: '',
        pathMatch: 'full'
    }
];
