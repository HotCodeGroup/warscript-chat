package main

import (
	"encoding/json"
)

type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// MessageFromClient simple anon message
type MessageFromClient struct {
	Message string `json:"message"`
}

// MessageToClient simple anon message
type MessageToClient struct {
	Message string `json:"message"`
}

type SignedMessage struct {
	Author  string `json:"author"`
	Message string `json:"message"`
}

type MessageHistoryQuery struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type MessagesResp struct {
	Messages []*SignedMessage `json:"messages"`
}

// WSObject object for ws connections
type WSObject struct {
	client  *Client
	Type    string          `json:"type"`
	ChatID  int64           `json:"chat_id"`
	Author  *UserInfo       `json:"author"`
	Payload json.RawMessage `json:"payload"`
}
