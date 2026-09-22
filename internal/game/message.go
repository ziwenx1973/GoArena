package game

import "encoding/json"

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}
type ClientMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
type GameEvent struct {
	PlayerID uint
	Choice   Choice
	Round    int
}
