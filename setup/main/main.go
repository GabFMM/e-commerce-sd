package main

import (
	"os"
	"log"
	"e-commerce-sd/setup"
	"e-commerce-sd/data"
	"e-commerce-sd/seguranca"
)

func main() {
	args := os.Args

	if len(os.Args) < 2 {
		log.Panic("[ERRO] Esperava 1 argumento na linha de comando, mas obteve 0")
	}

	switch args[1] {
	case "iniciar":
		setup.ConfigurarFilas()
		seguranca.GerarChaves()
		data.IniciarBanco()
	case "encerrar":
		data.EncerrarBanco()
	default:
		log.Panic("[ERRO] Argumento na linha de comando é inválido")
	}

	log.Println("[LOG] Setup concluído com sucesso.")
}
