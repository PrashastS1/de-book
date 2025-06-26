package main

import (
	"encoding/json"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// Book represents a book being offered for exchange.
type Book struct {
	Owner     peer.ID   `json:"owner"`
	ISBN      string    `json:"isbn"`
	Title     string    `json/:"title"` // Example field
	Author    string    `json:"author"` // Example field
	Timestamp time.Time `json:"timestamp"`
	// In the future, this will also contain a cryptographic signature
}

// Marshal converts a Book object to its JSON byte representation.
func (b *Book) Marshal() ([]byte, error) {
	return json.Marshal(b)
}

// Unmarshal fills a Book object from its JSON byte representation.
func (b *Book) Unmarshal(data []byte) error {
	return json.Unmarshal(data, b)
}