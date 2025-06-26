package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
	"sync"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// Block represents a single entry in our ledger (a "transaction").
type Block struct {
	Index     int
	Timestamp string
	Data      string 
	PrevHash  string
	Hash      string
	Signature []byte
	CreatorPubKey crypto.PubKey
}

var Blockchain []Block
var bcMutex = &sync.Mutex{}

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
	if b.CreatorPubKey == nil {
		return false, fmt.Errorf("public key is nil")
	}
	hash := b.calculateHash()
	return b.CreatorPubKey.Verify([]byte(hash), b.Signature)
}

// isBlockValid checks if a new block is valid to be added to the chain.
func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		fmt.Println("Invalid index")
		return false
	}
	if oldBlock.Hash != newBlock.PrevHash {
		fmt.Println("Invalid previous hash")
		return false
	}
	if newBlock.calculateHash() != newBlock.Hash {
		fmt.Println("Invalid hash")
		return false
	}
	valid, err := newBlock.verifySignature()
	if err != nil || !valid {
		fmt.Printf("Invalid signature: %v\n", err)
		return false
	}
	return true
}

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

func generateBlock(privKey crypto.PrivKey, transaction Transaction) (Block, error) {
	// We need to lock here because we are reading the last block
	bcMutex.Lock()
	oldBlock := Blockchain[len(Blockchain)-1]
	bcMutex.Unlock()

	txBytes, err := json.Marshal(transaction)
	if err != nil {
		return Block{}, err
	}

	newBlock := Block{
		Index:     oldBlock.Index + 1,
		Timestamp: time.Now().String(),
		Data:      string(txBytes),
		PrevHash:  oldBlock.Hash,
	}
	newBlock.Hash = newBlock.calculateHash()
	
	if err := newBlock.signBlock(privKey); err != nil {
		return Block{}, err
	}

	return newBlock, nil
}