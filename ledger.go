package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// Block represents a single entry in our ledger (a "transaction").
type Block struct {
	Index     int
	Timestamp string
	// The actual data being stored. For now, a simple string.
	// We'll make this more structured later.
	Data      string 
	PrevHash  string
	Hash      string
	// The signature proves who created this block.
	Signature []byte
	// The public key of the creator.
	CreatorPubKey crypto.PubKey
}

// Blockchain is a series of validated Blocks.
var Blockchain []Block

// calculateHash creates a SHA256 hash of a Block.
func (b *Block) calculateHash() string {
	record := fmt.Sprintf("%d%s%s%s", b.Index, b.Timestamp, b.Data, b.PrevHash)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// signBlock signs the block's hash with the creator's private key.
func (b *Block) signBlock(privKey crypto.PrivKey) error {
	hash := b.calculateHash()
	sig, err := privKey.Sign([]byte(hash))
	if err != nil {
		return err
	}
	b.Signature = sig
	b.CreatorPubKey = privKey.GetPublic()
	return nil
}

// verifySignature checks if the block's signature is valid.
func (b *Block) verifySignature() (bool, error) {
	hash := b.calculateHash()
	return b.CreatorPubKey.Verify([]byte(hash), b.Signature)
}

// isBlockValid checks if a new block is valid to be added to the chain.
func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}
	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}
	if newBlock.calculateHash() != newBlock.Hash {
		return false
	}
	// Verify the signature
	valid, err := newBlock.verifySignature()
	if err != nil || !valid {
		return false
	}
	return true
}

// createGenesisBlock creates the very first block in the chain.
func createGenesisBlock() {
	genesisBlock := Block{
		Index:     0,
		Timestamp: time.Now().String(),
		Data:      "Genesis Block",
		PrevHash:  "",
	}
	genesisBlock.Hash = genesisBlock.calculateHash()
	Blockchain = append(Blockchain, genesisBlock)
}