package main

import (
	"log"
	"e-commerce-sd/setup"
)

func main() {
	setup.ConfigurarFilas()
	setup.IniciarBanco()
	log.Println("[LOG] Setup concluído com sucesso.")
}
