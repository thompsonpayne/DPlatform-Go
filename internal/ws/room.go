package ws

import (
	"context"
)

type Room struct {
	id         string
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client

	manager *RoomManager
}

func NewRoom(id string, manager *RoomManager) *Room {
	return &Room{
		id:         id,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),

		manager: manager,
	}
}

func (h *Room) Run(ctx context.Context) {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
		case msg := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// remove slow clients
					delete(h.clients, c)
					close(c.send)
				}
			}
		case <-ctx.Done():
			for c := range h.clients {
				delete(h.clients, c)
				close(c.send)
			}
			return

		}
	}
}
