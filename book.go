package main

import (
	"encoding/json"
)

// Book represents a book being offered for exchange.
type Book struct {
	ISBN      string    `json:"isbn"`
	Title     string    `json:"title"` // Example field
	Author    string    `json:"author"` // Example field
}

type Transaction struct {
	Type string `json:"type"` // e.g., "REGISTER_BOOK"
	Book Book   `json:"book"`
}

// Marshal converts a Book object to its JSON byte representation.
func (t *Transaction) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

func (t *Transaction) Unmarshal(data []byte) error {
	return json.Unmarshal(data, t)
}