package main

import (
	"encoding/json"
	"log"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan WSObject
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan WSObject),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case inRaw := <-h.broadcast:
			var outRaw WSObject
			ok := true
			switch inRaw.Type {
			case "message":
				var inMsg MessageFromClient
				err := json.Unmarshal(inRaw.Payload, &inMsg)
				if err != nil {
					log.Printf("data in WSObject does not corresponds to type message: %v", err)
					ok = false
				}

				payload, _ := json.Marshal(&MessageToClient{
					Message: inMsg.Message,
				})

				outRaw = WSObject{
					Type:    "message",
					Payload: payload,
				}
			default:
				log.Printf("unknown type in WSObject: `%s`", inRaw.Type)
				ok = false
			}
			if ok {
				for client := range h.clients {
					select {
					case client.send <- outRaw:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
		}
	}
}
