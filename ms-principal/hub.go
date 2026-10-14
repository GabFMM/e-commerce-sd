package main

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Hub de SSE: cada conexão aberta no /api/eventos recebe o seu próprio canal,
// e todo evento publicado via broadcast() é entregue a TODAS as conexões.
// (PayloadEvento está definido no main.go, mesmo package.)
type hub struct {
	mu   sync.RWMutex
	subs map[chan PayloadEvento]struct{}
}

var sseHub = &hub{
	subs: make(map[chan PayloadEvento]struct{}),
}

// Registra uma nova conexão SSE e devolve o canal dela.
func (h *hub) subscribe() chan PayloadEvento {
	ch := make(chan PayloadEvento, 20)

	h.mu.Lock()
	h.subs[ch] = struct{}{}
	total := len(h.subs)
	h.mu.Unlock()

	log.Printf("[SSE] Cliente conectado. Conexões abertas: %d\n", total)
	return ch
}

// Remove a conexão (chamado quando o navegador fecha a aba/conexão).
func (h *hub) unsubscribe(ch chan PayloadEvento) {
	h.mu.Lock()
	delete(h.subs, ch)
	total := len(h.subs)
	h.mu.Unlock()

	log.Printf("[SSE] Cliente desconectado. Conexões abertas: %d\n", total)
}

// Entrega o evento a todas as conexões abertas, sem nunca bloquear:
// se o buffer de um cliente lento estiver cheio, o evento é descartado só para ele.
func (h *hub) broadcast(ev PayloadEvento) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.subs) == 0 {
		log.Printf("[SSE] Nenhum cliente conectado. Evento '%s' do pedido %d descartado.\n", ev.Tipo, ev.PedidoID)
		return
	}

	for ch := range h.subs {
		select {
		case ch <- ev:
		default:
			log.Printf("[SSE] Buffer cheio para um cliente. Evento '%s' do pedido %d descartado para ele.\n", ev.Tipo, ev.PedidoID)
		}
	}
}

// Handler do GET /api/eventos
func streamEventos(c *gin.Context) {
	h := c.Writer.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")

	ch := sseHub.subscribe()
	defer sseHub.unsubscribe(ch)

	// Flush inicial: faz o navegador disparar o onopen do EventSource imediatamente
	fmt.Fprint(c.Writer, ": conectado\n\n")
	c.Writer.Flush()

	// Comentário periódico para manter a conexão viva (proxies/navegadores)
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			// Cliente desconectou/fechou a aba
			return false
		case ev := <-ch:
			c.SSEvent("message", ev)
			return true
		case <-ticker.C:
			fmt.Fprint(w, ": ping\n\n")
			return true
		}
	})
}
