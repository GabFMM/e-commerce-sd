package main

import (
	"os"
	"log"
	"e-commerce-sd/setup"
)

func main() {
	args := os.Args

	if len(os.Args) == 0 {
		log.Panic("[ERRO] Esperava 1 argumento na linha de comando, mas obteve 0")
		return
	}

	setup.ConfigurarFilas()

	switch args[1] {
	case "iniciar":
		setup.IniciarBanco()
	case "encerrar":
		setup.EncerrarBanco()
	}

	log.Println("[LOG] Setup concluído com sucesso.")
}
