package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
	// "github.com/libp2p/go-libp2p/core/peer"
)

// ... (constants and bookRegistry are the same) ...
const DeBookTopic = "/de-book/1.0.0"
const DiscoveryTag = "de-book-discovery"
var bookRegistry = make(map[string]Book)

// setupDiscoveryAndPubSub initializes the pubsub system and starts peer discovery.
func setupDiscoveryAndPubSub(ctx context.Context, node host.Host, kadDHT *dht.IpfsDHT) (*pubsub.Topic, error) {
	ps, err := pubsub.NewGossipSub(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	topic, err := ps.Join(DeBookTopic)
	if err != nil {
		return nil, fmt.Errorf("failed to join topic: %w", err)
	}

	// 3. Start a goroutine to read messages from the topic
	// --- MODIFICATION HERE ---
	go readFromTopic(ctx, node, topic)
	go findPeers(ctx, node, routing.NewRoutingDiscovery(kadDHT))

	// 4. Set up peer discovery
	util.Advertise(ctx, routing.NewRoutingDiscovery(kadDHT), DiscoveryTag)
	fmt.Println("Successfully advertised our presence on the network!")

	return topic, nil
}

// func readFromTopic(ctx context.Context, node host.Host, topic *pubsub.Topic) {
// 	sub, err := topic.Subscribe()
// 	if err != nil {
// 		fmt.Println("Error subscribing to topic:", err)
// 		return
// 	}
// 	defer sub.Cancel()

// 	for {
// 		msg, err := sub.Next(ctx)
// 		if err != nil { return }

// 		if msg.ReceivedFrom == node.ID() { continue }

// 		// Unmarshal the data into a Block
// 		var newBlock Block
// 		if err := json.Unmarshal(msg.Data, &newBlock); err != nil {
// 			fmt.Println("Error unmarshaling block data:", err)
// 			continue
// 		}

// 		// *** THE CRITICAL VALIDATION STEP ***
// 		// We could lock the blockchain here to prevent race conditions
// 		// For now, we'll keep it simple.
// 		lastBlock := Blockchain[len(Blockchain)-1]
// 		if isBlockValid(newBlock, lastBlock) {
// 			Blockchain = append(Blockchain, newBlock)
// 			fmt.Printf("\n--- Valid New Block Received ---\n")
// 			fmt.Printf("From: %s\n", msg.ReceivedFrom.ShortString())
// 			fmt.Printf("Data: %s\n", newBlock.Data)
// 			fmt.Printf("Appended to local ledger. Chain length: %d\n", len(Blockchain))
// 			fmt.Printf("------------------------------\n> ")
// 		} else {
// 			fmt.Println("Received an invalid block. Discarding.")
// 		}
// 	}
// }

// readFromTopic now uses a temporary struct for unmarshaling.
func readFromTopic(ctx context.Context, node host.Host, topic *pubsub.Topic) {
	sub, err := topic.Subscribe()
	if err != nil {
		fmt.Println("Error subscribing to topic:", err)
		return
	}
	defer sub.Cancel()

	for {
		msg, err := sub.Next(ctx)
		if err != nil { return }
		if msg.ReceivedFrom == node.ID() { continue }

		// Define the temporary struct for JSON decoding
		type jsonBlock struct {
			Index     int    `json:"Index"`
			Timestamp string `json:"Timestamp"`
			Data      string `json:"Data"`
			PrevHash  string `json:"PrevHash"`
			Hash      string `json:"Hash"`
			// --- THE FIX IS HERE: Expect strings, not byte slices ---
			Signature     string `json:"Signature"`
			CreatorPubKey string `json:"CreatorPubKey"`
		}

		// 1. Unmarshal into the temporary struct with string fields
		var jb jsonBlock
		if err := json.Unmarshal(msg.Data, &jb); err != nil {
			fmt.Println("Error unmarshaling jsonBlock data:", err)
			continue
		}

		// 2. Base64-decode the strings back into byte slices
		sigBytes, err := base64.StdEncoding.DecodeString(jb.Signature)
		if err != nil {
			fmt.Println("Error decoding signature:", err)
			continue
		}
		pubKeyBytes, err := base64.StdEncoding.DecodeString(jb.CreatorPubKey)
		if err != nil {
			fmt.Println("Error decoding public key:", err)
			continue
		}
		
		// 3. Convert the raw public key bytes back into a crypto.PubKey object
		pubKey, err := crypto.UnmarshalPublicKey(pubKeyBytes)
		if err != nil {
			fmt.Println("Error unmarshaling public key:", err)
			continue
		}
		
		// 4. Manually construct the real Block
		newBlock := Block{
			Index:         jb.Index,
			Timestamp:     jb.Timestamp,
			Data:          jb.Data,
			PrevHash:      jb.PrevHash,
			Hash:          jb.Hash,
			Signature:     sigBytes, // Use the decoded byte slice
			CreatorPubKey: pubKey,
		}

		// Validation step with mutex remains the same
		bcMutex.Lock()
		if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
			Blockchain = append(Blockchain, newBlock)
			fmt.Printf("\n--- Valid New Block Received ---\n")
			fmt.Printf("From: %s\n", msg.ReceivedFrom.ShortString())
			fmt.Printf("Data: %s\n", newBlock.Data)
			fmt.Printf("Appended to local ledger. Chain length: %d\n", len(Blockchain))
			fmt.Printf("------------------------------\n> ")
		} else {
			fmt.Println("Received an invalid block. Discarding.")
		}
		bcMutex.Unlock()
	}
}

// findPeers function remains the same.
func findPeers(ctx context.Context, node host.Host, discovery *routing.RoutingDiscovery) {
	ticker := time.NewTicker(1 * time.Minute) 
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Println("Searching for other De-Book peers...")
			peerChan, err := discovery.FindPeers(ctx, DiscoveryTag)
			if err != nil {
				fmt.Println("Failed to find peers:", err)
				continue
			}

			for p := range peerChan {
				if p.ID == node.ID() || len(p.Addrs) == 0 {
					continue 
				}
				
				if err := node.Connect(ctx, p); err != nil {
					// fmt.Printf("Failed to connect to discovered peer %s: %s\n", p.ID.ShortString(), err)
				} else {
					fmt.Printf("Connected to a new De-Book peer: %s\n", p.ID.ShortString())
				}
			}
		}
	}
}