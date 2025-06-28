package main

import (
	"encoding/json"
)

// Book represents a book being offered for exchange.
type Book struct {
	ISBN      string    `json:"isbn"`
	Title     string    `json:"title"` // Example field
	Author    string    `json:"author"` // Example field
	CoverCID  string `json:"coverCid,omitempty"`
}

type TradeProposal struct {
	ProposerID  string `json:"proposerId"`
	ProposerBookISBN string `json:"proposerBookIsbn"`
	TargetID    string `json:"targetId"`
	TargetBookISBN   string `json:"targetBookIsbn"`
	ProposalID  string `json:"proposalId"`
}

type Transaction struct {
	Type    string        `json:"type"` // "REGISTER_BOOK", "PROPOSE_TRADE", "CONFIRM_TRADE"
	Book    Book          `json:"book,omitempty"`
	Trade   TradeProposal `json:"trade,omitempty"`
	// The ProposalID that a confirmation refers to.
	ProposalID string        `json:"proposalId,omitempty"` 
}

// Marshal converts a Book object to its JSON byte representation.
func (t *Transaction) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

func (t *Transaction) Unmarshal(data []byte) error {
	return json.Unmarshal(data, t)
}