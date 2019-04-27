package main

import (
	"encoding/json"
)

// MessageFromClient simple anon message
type MessageFromClient struct {
	Message string `json:"message"`
}

// MessageToClient simple anon message
type MessageToClient struct {
	Message string `json:"message"`
}

// WSObject object for ws connections
type WSObject struct {
	Type    string          `json:"type"`
	Author  string          `json:"author"`
	Payload json.RawMessage `json:"payload"`
}
