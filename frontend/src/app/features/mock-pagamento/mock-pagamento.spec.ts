import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MockPagamento } from './mock-pagamento';

describe('MockPagamento', () => {
  let component: MockPagamento;
  let fixture: ComponentFixture<MockPagamento>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MockPagamento],
    }).compileComponents();

    fixture = TestBed.createComponent(MockPagamento);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
