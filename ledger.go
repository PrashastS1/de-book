package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
)

type Block struct {
	Index         int
	Timestamp     string
	Data          string
	PrevHash      string
	Hash          string
	Signature     []byte
	CreatorPubKey crypto.PubKey
}

type SerializableBlock struct {
	Index         int    `json:"Index"`
	Timestamp     string `json:"Timestamp"`
	Data          string `json:"Data"`
	PrevHash      string `json:"PrevHash"`
	Hash          string `json:"Hash"`
	Signature     []byte `json:"Signature"`     // json handles []byte by base64-encoding it
	CreatorPubKey []byte `json:"CreatorPubKey"` // We will store the raw bytes of the public key
}

func (b *Block) toSerializable() (SerializableBlock, error) {
	var pubKeyBytes []byte
	var err error
	if b.CreatorPubKey != nil {
		pubKeyBytes, err = crypto.MarshalPublicKey(b.CreatorPubKey)
		if err != nil {
			return SerializableBlock{}, err
		}
	}

	return SerializableBlock{
		Index:         b.Index,
		Timestamp:     b.Timestamp,
		Data:          b.Data,
		PrevHash:      b.PrevHash,
		Hash:          b.Hash,
		Signature:     b.Signature,
		CreatorPubKey: pubKeyBytes, // This will be nil for the Genesis Block, which is fine.
	}, nil
}

func (sb *SerializableBlock) toBlock() (Block, error) {
	var pubKey crypto.PubKey
	var err error
	if sb.CreatorPubKey != nil {
		pubKey, err = crypto.UnmarshalPublicKey(sb.CreatorPubKey)
		if err != nil {
			return Block{}, err
		}
	}

	return Block{
		Index:         sb.Index,
		Timestamp:     sb.Timestamp,
		Data:          sb.Data,
		PrevHash:      sb.PrevHash,
		Hash:          sb.Hash,
		Signature:     sb.Signature,
		CreatorPubKey: pubKey, // Will be nil for Genesis Block.
	}, nil
}

var Blockchain []Block
var bcMutex = &sync.Mutex{}

func (b *Block) calculateHash() string {
	record := fmt.Sprintf("%d%s%s%s", b.Index, b.Timestamp, b.Data, b.PrevHash)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

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

func (b *Block) verifySignature() (bool, error) {
	if b.CreatorPubKey == nil {
		return false, fmt.Errorf("public key is nil")
	}
	hash := b.calculateHash()
	return b.CreatorPubKey.Verify([]byte(hash), b.Signature)
}

// func isBlockValid(newBlock, oldBlock Block) bool {
// 	if oldBlock.Index+1 != newBlock.Index {
// 		fmt.Println("Invalid index")
// 		return false
// 	}
// 	if oldBlock.Hash != newBlock.PrevHash {
// 		fmt.Println("Invalid previous hash")
// 		return false
// 	}
// 	if newBlock.calculateHash() != newBlock.Hash {
// 		fmt.Println("Invalid hash")
// 		return false
// 	}

// 	if newBlock.Index > 0 {
// 		valid, err := newBlock.verifySignature()
// 		if err != nil || !valid {
// 			fmt.Printf("Invalid signature on block %d: %v\n", newBlock.Index, err)
// 			return false
// 		}
// 	}

// 	return true
// }

func createGenesisBlock() {
	genesisBlock := Block{
		Index:     0,
		Timestamp: "2024-01-01T00:00:00Z", // A fixed, constant timestamp
		Data:      "Genesis Block",
		PrevHash:  "",
		// Signature and CreatorPubKey are nil, as they should be.
	}
	genesisBlock.Hash = genesisBlock.calculateHash()
	Blockchain = append(Blockchain, genesisBlock)
}

func generateBlock(privKey crypto.PrivKey, transaction Transaction) (Block, error) {
	bcMutex.Lock()
	oldBlock := Blockchain[len(Blockchain)-1]
	bcMutex.Unlock()

	txBytes, err := json.Marshal(transaction)
	if err != nil {
		return Block{}, err
	}

	newBlock := Block{
		Index:     oldBlock.Index + 1,
		Timestamp: time.Now().UTC().Format(time.RFC3339), // Use current time for new blocks
		Data:      string(txBytes),
		PrevHash:  oldBlock.Hash,
	}
	newBlock.Hash = newBlock.calculateHash()

	if err := newBlock.signBlock(privKey); err != nil {
		return Block{}, err
	}

	return newBlock, nil
}


func isChainValid(chain []Block) bool {
	// Create a temporary Genesis Block with the exact same constant values.
	expectedGenesis := Block{
		Index:     0,
		Timestamp: "2024-01-01T00:00:00Z", // Use the same constant
		Data:      "Genesis Block",
		PrevHash:  "",
	}
	expectedGenesis.Hash = expectedGenesis.calculateHash()

	// Compare the hash of the received genesis block with our expected hash.
	if chain[0].Hash != expectedGenesis.Hash {
		fmt.Println("Invalid Genesis Block in received chain.")
		return false
	}
	
	// The rest of the validation is the same.
	for i := 1; i < len(chain); i++ {
		if !isBlockValid(chain[i], chain[i-1]) {
			return false
		}
	}
	return true
}

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
	if newBlock.Index > 0 {
		valid, err := newBlock.verifySignature()
		if err != nil || !valid {
			return false
		}
	}
	return true
}