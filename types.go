package main

import (
	"encoding/json"
)

type MessageFromClient struct {
	Message string `json:"message"`
}

type MessageToClient struct {
	Message string `json:"message"`
}

type WSObject struct {
	Type    string          `json:"string"`
	Payload json.RawMessage `json:"payload"`
}
