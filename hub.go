package main

import (
	"encoding/json"

	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"
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

func (h *Hub) registerClient(client *Client) {
	for _, chatID := range client.conversations {
		if _, ok := h.clients[chatID]; !ok {
			h.clients[chatID] = make(map[*Client]bool, 0)
		}

		h.clients[chatID][client] = true
	}
}

func (h *Hub) unregisterClient(client *Client) {
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
}

func (h *Hub) processMessage(inRawMsg *WSObject) error {
	var inMsg MessageFromClient
	err := json.Unmarshal(inRawMsg.Payload, &inMsg)
	if err != nil {
		return errors.Wrap(err, "data in WSObject does not corresponds to type message")
	}

	model := &MessageModel{
		Message: pgtype.Text{String: inMsg.Message, Status: pgtype.Present},
		ConvID:  pgtype.Int8{Int: inRawMsg.ChatID, Status: pgtype.Present},
	}

	if inRawMsg.Author != nil {
		model.Author = pgtype.Text{String: inRawMsg.Author.Username, Status: pgtype.Present}
	} else {
		model.Author = pgtype.Text{String: "", Status: pgtype.Null}
	}

	err = Messages.Create(model)
	if err != nil {
		return errors.Wrap(err, "can not save message")
	}

	payload, _ := json.Marshal(&MessageToClient{
		ID:      model.ID.Int,
		Message: inMsg.Message,
	})

	outRaw := WSObject{
		Type:    "message",
		ChatID:  inRawMsg.ChatID,
		Author:  inRawMsg.Author,
		Payload: payload,
	}

	for client := range h.clients[outRaw.ChatID] {
		h.sendMessage(client, outRaw)
	}

	return nil
}

func (h *Hub) processHistory(inRawMsg *WSObject) error {
	var inMsg MessageHistoryQuery
	err := json.Unmarshal(inRawMsg.Payload, &inMsg)
	if err != nil {
		return errors.Wrap(err, "data in WSObject does not corresponds to type message")
	}

	msgs, err := Messages.GetMessagesByConvID(inRawMsg.ChatID, inMsg.Limit, inMsg.Offset)
	if err != nil {
		return errors.Wrap(err, "can not get messages")
	}

	resp := MessagesResp{
		Messages: make([]*SignedMessage, 0, 0),
	}
	for _, msg := range msgs {
		resp.Messages = append(resp.Messages, &SignedMessage{
			ID:      msg.ID.Int,
			Message: msg.Message.String,
			Author:  msg.Author.String,
		})
	}

	payload, _ := json.Marshal(&resp)
	outRaw := WSObject{
		Type:    "messages",
		ChatID:  inRawMsg.ChatID,
		Author:  inRawMsg.Author,
		Payload: payload,
	}

	h.sendMessage(inRawMsg.client, outRaw)
	return nil
}

func (h *Hub) sendMessage(to *Client, msg WSObject) {
	select {
	case to.send <- msg:
	default:
		close(to.send)
		delete(h.clients[msg.ChatID], to)

		if len(h.clients[msg.ChatID]) == 0 {
			delete(h.clients, msg.ChatID)
		}
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case inRaw := <-h.broadcast:
			var err error
			switch inRaw.Type {
			case "message":
				err = h.processMessage(&inRaw)
			case "messages":
				err = h.processHistory(&inRaw)
			default:
				err = errors.New("unknown type in WSObject: " + inRaw.Type)
			}

			if err != nil {
				logger.Errorf("message process error: %s", err)
				break
			}
		}
	}
}
