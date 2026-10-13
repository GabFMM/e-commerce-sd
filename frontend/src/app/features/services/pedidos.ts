import { HttpClient } from '@angular/common/http';
import { inject, Service } from '@angular/core';

@Service()
export class PedidosService {
    private http = inject(HttpClient);
    private apiUrl = 'http://localhost:8080';

    
}
