import { ComponentFixture, TestBed } from '@angular/core/testing';
import { CarrinhoService } from './carrinho';

describe('Carrinho', () => {
  let component: CarrinhoService;
  let fixture: ComponentFixture<CarrinhoService>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CarrinhoService],
    }).compileComponents();

    fixture = TestBed.createComponent(CarrinhoService);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
