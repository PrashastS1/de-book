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

// Block represents a single entry in our ledger (a "transaction").
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

// func (b *Block) toSerializable() (SerializableBlock, error) {
// 	pubKeyBytes, err := crypto.MarshalPublicKey(b.CreatorPubKey)
// 	if err != nil {
// 		return SerializableBlock{}, err
// 	}
// 	return SerializableBlock{
// 		Index:         b.Index,
// 		Timestamp:     b.Timestamp,
// 		Data:          b.Data,
// 		PrevHash:      b.PrevHash,
// 		Hash:          b.Hash,
// 		Signature:     b.Signature,
// 		CreatorPubKey: pubKeyBytes,
// 	}, nil
// }

func (b *Block) toSerializable() (SerializableBlock, error) {
    // --- THE FIX IS HERE ---
	// Handle the Genesis Block, which has no public key.
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

// func (sb *SerializableBlock) toBlock() (Block, error) {
// 	pubKey, err := crypto.UnmarshalPublicKey(sb.CreatorPubKey)
// 	if err != nil {
// 		return Block{}, err
// 	}
// 	return Block{
// 		Index:         sb.Index,
// 		Timestamp:     sb.Timestamp,
// 		Data:          sb.Data,
// 		PrevHash:      sb.PrevHash,
// 		Hash:          sb.Hash,
// 		Signature:     sb.Signature,
// 		CreatorPubKey: pubKey,
// 	}, nil
// }

// fromSerializable converts a simple, serializable block back to a rich Block.
func (sb *SerializableBlock) toBlock() (Block, error) {
    // --- ADD A NIL CHECK HERE TOO ---
	var pubKey crypto.PubKey
	var err error
	// Only unmarshal if the key bytes are not nil.
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

	if newBlock.Index > 0 {
		valid, err := newBlock.verifySignature()
		if err != nil || !valid {
			fmt.Printf("Invalid signature on block %d: %v\n", newBlock.Index, err)
			return false
		}
	}

	return true
	// valid, err := newBlock.verifySignature()
	// if err != nil || !valid {
	// 	fmt.Printf("Invalid signature: %v\n", err)
	// 	return false
	// }
	// return true
}

// func createGenesisBlock() {
// 	genesisBlock := Block{
// 		Index:     0,
// 		Timestamp: time.Now().String(),
// 		Data:      "Genesis Block",
// 		PrevHash:  "",
// 	}
// 	genesisBlock.Hash = genesisBlock.calculateHash()
// 	Blockchain = append(Blockchain, genesisBlock)
// }

func createGenesisBlock() {
	genesisBlock := Block{
		Index:     0,
		Timestamp: "2023-01-01T00:00:00Z", // A fixed timestamp
		Data:      "Genesis Block",
		PrevHash:  "",
		// Signature and CreatorPubKey are nil, correctly representing the root.
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

// func isChainValid(chain []Block) bool {
// 	// The first block must be a genesis block (simplified check)
// 	if chain[0].Index != 0 {
// 		return false
// 	}
// 	// Validate every subsequent block in the chain
// 	for i := 1; i < len(chain); i++ {
// 		if !isBlockValid(chain[i], chain[i-1]) {
// 			return false
// 		}
// 	}
// 	return true
// }

func isChainValid(chain []Block) bool {
	// The first block must be a genesis block (check its hash)
	// Let's create a temporary one to get the expected hash.
	expectedGenesis := Block{
		Index:     0,
		Timestamp: "2023-01-01T00:00:00Z",
		Data:      "Genesis Block",
		PrevHash:  "",
	}
	expectedGenesis.Hash = expectedGenesis.calculateHash()

	if chain[0].Hash != expectedGenesis.Hash {
		fmt.Println("Invalid Genesis Block in received chain.")
		return false
	}
    // ... rest of the function is the same ...
	for i := 1; i < len(chain); i++ {
		if !isBlockValid(chain[i], chain[i-1]) {
			return false
		}
	}
	return true
}