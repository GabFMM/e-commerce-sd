package main

import (
	"e-commerce-sd/seguranca"
	"e-commerce-sd/setup"
	"log"
)

func main() {
	setup.ConfigurarFilas()
	
	seguranca.GerarChaves()

	log.Println("[LOG] Setup concluído com sucesso.")
}
