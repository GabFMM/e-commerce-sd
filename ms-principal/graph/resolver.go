package graph

import "e-commerce-sd/ms-principal/dto"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	// Publicar[...] é fornecida pelo main, que possui o canal RabbitMQ
	// e a chave privada usados para publicar o evento.
	PublicarPedidoCriado func(dto.PedidoCriadoDTO) error
	PublicarPedidoExcluido func(dto.PedidoDTO) error
}
