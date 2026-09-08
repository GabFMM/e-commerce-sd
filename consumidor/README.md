1. Como Rodar, primeiro ligar o microsserviço
```
#Terminal 1
 go run ./ms-promocoes/main.go 
```

2. Depois ligar cada consumidor em sua fila em terminais diferentes
```
#Terminal 2
go run consumidor/main.go Q1

#Terminal 3
go run consumidor/main.go Q2
```