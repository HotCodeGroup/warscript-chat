package main

import (
	"encoding/json"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	clients    map[int64]map[*Client]bool
	broadcast  chan WSObject
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan WSObject),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[int64]map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			for _, chatID := range client.conversations {
				if _, ok := h.clients[chatID]; !ok {
					h.clients[chatID] = make(map[*Client]bool, 0)
				}

				h.clients[chatID][client] = true
			}

		case client := <-h.unregister:
			for _, chatID := range client.conversations {
				if _, ok := h.clients[chatID]; ok {
					if _, ok := h.clients[chatID][client]; ok {
						delete(h.clients[chatID], client)
						close(client.send)
					}

					if len(h.clients[chatID]) == 0 {
						delete(h.clients, chatID)
					}
				}
			}
		case inRaw := <-h.broadcast:
			var outRaw WSObject
			ok := true
			switch inRaw.Type {
			case "message":
				var inMsg MessageFromClient
				err := json.Unmarshal(inRaw.Payload, &inMsg)
				if err != nil {
					logger.Errorf("data in WSObject does not corresponds to type message: %v", err)
					ok = false
				}

				payload, _ := json.Marshal(&MessageToClient{
					Message: inMsg.Message,
				})

				outRaw = WSObject{
					Type:    "message",
					ChatID:  inRaw.ChatID,
					Author:  inRaw.Author,
					Payload: payload,
				}
			default:
				logger.Warnf("unknown type in WSObject: `%s`", inRaw.Type)
				ok = false
			}
			if ok {
				for client := range h.clients[outRaw.ChatID] {
					select {
					case client.send <- outRaw:
					default:
						close(client.send)
						delete(h.clients[outRaw.ChatID], client)

						if len(h.clients[outRaw.ChatID]) == 0 {
							delete(h.clients, outRaw.ChatID)
						}
					}
				}
			}
		}
	}
}
