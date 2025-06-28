package main

import "encoding/json"

const (
	MsgTypeRequestChain  = "REQUEST_CHAIN"
	MsgTypeRespondChain  = "RESPOND_CHAIN"
	MsgTypeAnnounceBlock = "ANNOUNCE_BLOCK"
)

type Message struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}