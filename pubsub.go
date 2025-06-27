// // package main

// // import (
// // 	"context"
// // 	"encoding/json"
// // 	"fmt"
// // 	"time"

// // 	pubsub "github.com/libp2p/go-libp2p-pubsub"
// // 	"github.com/libp2p/go-libp2p/core/host"
// // 	"github.com/libp2p/go-libp2p/core/peer"
// // 	dht "github.com/libp2p/go-libp2p-kad-dht"
// // 	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
// // 	"github.com/libp2p/go-libp2p/p2p/discovery/util"
// // )

// // // ... (constants and bookRegistry are the same) ...
// // const DeBookTopic = "/de-book/1.0.0"
// // const DiscoveryTag = "de-book-discovery"
// // var bookRegistry = make(map[string]Book)

// // // setupDiscoveryAndPubSub initializes the pubsub system and starts peer discovery.
// // func setupDiscoveryAndPubSub(ctx context.Context, node host.Host, kadDHT *dht.IpfsDHT) (*pubsub.Topic, error) {
// // 	ps, err := pubsub.NewGossipSub(ctx, node)
// // 	if err != nil {
// // 		return nil, fmt.Errorf("failed to create pubsub: %w", err)
// // 	}

// // 	topic, err := ps.Join(DeBookTopic)
// // 	if err != nil {
// // 		return nil, fmt.Errorf("failed to join topic: %w", err)
// // 	}

// // 	// 3. Start a goroutine to read messages from the topic
// // 	// --- MODIFICATION HERE ---
// // 	go readFromTopic(ctx, node, topic)
// // 	go findPeers(ctx, node, routing.NewRoutingDiscovery(kadDHT))

// // 	// 4. Set up peer discovery
// // 	util.Advertise(ctx, routing.NewRoutingDiscovery(kadDHT), DiscoveryTag)
// // 	fmt.Println("Successfully advertised our presence on the network!")

// // 	return topic, nil
// // }

// // // readFromTopic now uses a temporary struct for unmarshaling.
// // func readFromTopic(ctx context.Context, node host.Host, topic *pubsub.Topic) {
// // 	sub, err := topic.Subscribe()
// // 	if err != nil {
// // 		fmt.Println("Error subscribing to topic:", err)
// // 		return
// // 	}
// // 	defer sub.Cancel()

// // 	for {
// // 		msg, err := sub.Next(ctx)
// // 		if err != nil { return }
// // 		if msg.ReceivedFrom == node.ID() { continue }

// // 		var message Message
// // 		if err := json.Unmarshal(msg.Data, &message); err != nil {
// // 			fmt.Println("Error unmarshaling generic message:", err)
// // 			continue
// // 		}

// // 		switch message.Type {
// // 		case MsgTypeAnnounceBlock:
// // 			handleAnnounceBlock(message.Payload)
// // 		// We will add handlers for the other types later
// // 		}

		
// // 	}
// // }

// // func handleAnnounceBlock(payload []byte) {
// // 	var sb SerializableBlock
// // 	if err := json.Unmarshal(payload, &sb); err != nil {
// // 		fmt.Println("Error unmarshaling announced block:", err)
// // 		return
// // 	}

// // 	newBlock, err := sb.toBlock()
// // 	if err != nil {
// // 		fmt.Println("Error converting to rich block:", err)
// // 		return
// // 	}

// // 	bcMutex.Lock()
// // 	defer bcMutex.Unlock()
	
// // 	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
// // 		Blockchain = append(Blockchain, newBlock)
// // 		fmt.Printf("\n--- Valid New Block Received & Appended ---\n")
// // 	} else {
// // 		// THIS IS THE KEY CHANGE!
// // 		// Our chain is out of sync. For now, let's just log it.
// // 		// In the next step, we'll request the full chain from the sender.
// // 		fmt.Println("Received a block that doesn't fit our chain (fork detected).")
// // 	}
// // }

// // // findPeers function remains the same.
// // func findPeers(ctx context.Context, node host.Host, discovery *routing.RoutingDiscovery) {
// // 	ticker := time.NewTicker(1 * time.Minute) 
// // 	defer ticker.Stop()

// // 	for {
// // 		select {
// // 		case <-ctx.Done():
// // 			return
// // 		case <-ticker.C:
// // 			fmt.Println("Searching for other De-Book peers...")
// // 			peerChan, err := discovery.FindPeers(ctx, DiscoveryTag)
// // 			if err != nil {
// // 				fmt.Println("Failed to find peers:", err)
// // 				continue
// // 			}

// // 			for p := range peerChan {
// // 				if p.ID == node.ID() || len(p.Addrs) == 0 {
// // 					continue 
// // 				}
				
// // 				if err := node.Connect(ctx, p); err != nil {
// // 					// fmt.Printf("Failed to connect to discovered peer %s: %s\n", p.ID.ShortString(), err)
// // 				} else {
// // 					fmt.Printf("Connected to a new De-Book peer: %s\n", p.ID.ShortString())
// // 				}
// // 			}
// // 		}
// // 	}
// // }

// package main

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"

// 	pubsub "github.com/libp2p/go-libp2p-pubsub"
// )

// // This will now be our main handler for incoming messages
// func handleTopicMessage(ctx context.Context, topic *pubsub.Topic, msg *pubsub.Message) {
// 	if msg.ReceivedFrom == topic.Host().ID() {
// 		return
// 	}

// 	var message Message
// 	if err := json.Unmarshal(msg.Data, &message); err != nil {
// 		fmt.Println("Error unmarshaling generic message:", err)
// 		return
// 	}

// 	switch message.Type {
// 	case MsgTypeAnnounceBlock:
// 		var sb SerializableBlock
// 		if err := json.Unmarshal(message.Payload, &sb); err != nil { return }
// 		newBlock, err := sb.toBlock()
// 		if err != nil { return }

// 		bcMutex.Lock()
// 		myLastBlock := Blockchain[len(Blockchain)-1]
// 		bcMutex.Unlock()

// 		// Two cases: The new block fits, or it doesn't.
// 		if newBlock.PrevHash == myLastBlock.Hash {
// 			bcMutex.Lock()
// 			if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
// 				Blockchain = append(Blockchain, newBlock)
// 				fmt.Printf("\n--- Appended new block %d ---\n", newBlock.Index)
// 			}
// 			bcMutex.Unlock()
// 		} else if newBlock.Index > myLastBlock.Index {
// 			// Their chain is longer than ours. We are out of date.
// 			fmt.Println("Our blockchain is behind. Requesting full chain...")
			
// 			// Send a request for the full chain *directly to the peer who sent the block*
// 			// NOTE: This direct message sending is an advanced topic.
// 			// For now, we'll broadcast the request.
// 			req := Message{Type: MsgTypeRequestChain, Payload: nil}
// 			reqBytes, _ := req.Marshal()
// 			topic.Publish(ctx, reqBytes)
// 		}

// 	case MsgTypeRequestChain:
// 		// A peer is asking for our chain. Let's send it to them.
// 		bcMutex.Lock()
// 		// Convert our entire chain to the serializable format
// 		serializableChain := make([]SerializableBlock, len(Blockchain))
// 		for i, b := range Blockchain {
// 			serializableChain[i], _ = b.toSerializable()
// 		}
// 		chainBytes, _ := json.Marshal(serializableChain)
// 		bcMutex.Unlock()

// 		resp := Message{Type: MsgTypeRespondChain, Payload: chainBytes}
// 		respBytes, _ := resp.Marshal()
// 		topic.Publish(ctx, respBytes)
// 		fmt.Println("A peer requested our chain. Sent it.")

// 	case MsgTypeRespondChain:
// 		// We received a full chain from a peer. Let's validate and potentially replace ours.
// 		var theirChain []SerializableBlock
// 		if err := json.Unmarshal(message.Payload, &theirChain); err != nil {
// 			return
// 		}

// 		// Convert to rich blocks
// 		richChain := make([]Block, len(theirChain))
// 		for i, sb := range theirChain {
// 			richChain[i], err = sb.toBlock()
// 			if err != nil { return }
// 		}
		
// 		bcMutex.Lock()
// 		// A simple consensus rule: if their chain is longer and valid, replace ours.
// 		if len(richChain) > len(Blockchain) && isChainValid(richChain) {
// 			Blockchain = richChain
// 			fmt.Println("Replaced our chain with a longer, valid chain from the network.")
// 		}
// 		bcMutex.Unlock()
// 	}
// }

// // New function to validate an entire chain
// func isChainValid(chain []Block) bool {
// 	// Check genesis block (simplified)
// 	if chain[0].Index != 0 {
// 		return false
// 	}
// 	for i := 1; i < len(chain); i++ {
// 		if !isBlockValid(chain[i], chain[i-1]) {
// 			return false
// 		}
// 	}
// 	return true
// }

// func readFromTopic(ctx context.Context, topic *pubsub.Topic) {
// 	sub, err := topic.Subscribe()
// 	//...
// 	for {
// 		msg, err := sub.Next(ctx)
// 		// ...
// 		go handleTopicMessage(ctx, topic, msg)
// 	}
// }

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
)

// setupDiscoveryAndPubSub was missing. Here it is again.
func setupDiscoveryAndPubSub(ctx context.Context, node host.Host, kadDHT *dht.IpfsDHT) (*pubsub.Topic, error) {
	ps, err := pubsub.NewGossipSub(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

    // This now works because DeBookTopic is defined in main.go (in the same package)
	topic, err := ps.Join(DeBookTopic) 
	if err != nil {
		return nil, fmt.Errorf("failed to join topic: %w", err)
	}

	go readFromTopic(ctx, node, topic)

	routingDiscovery := routing.NewRoutingDiscovery(kadDHT)
    // This now works because DiscoveryTag is defined in main.go
	util.Advertise(ctx, routingDiscovery, DiscoveryTag) 
	fmt.Println("Successfully advertised our presence on the network!")
	go findPeers(ctx, node, routingDiscovery)

	return topic, nil
}


// handleTopicMessage is the new message router.
// We need to pass 'node' into it to get the local peer ID.
func handleTopicMessage(ctx context.Context, node host.Host, topic *pubsub.Topic, msg *pubsub.Message) {
	// Don't process our own messages
	if msg.ReceivedFrom == node.ID() {
		return
	}

	var message Message
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		fmt.Println("Error unmarshaling generic message:", err)
		return
	}

	switch message.Type {
	case MsgTypeAnnounceBlock:
		var sb SerializableBlock
		if err := json.Unmarshal(message.Payload, &sb); err != nil {
			fmt.Println("Error unmarshaling announced block:", err)
			return
		}
		newBlock, err := sb.toBlock()
		if err != nil {
			fmt.Println("Error converting block to rich type:", err)
			return
		}

		bcMutex.Lock()
		myLastBlock := Blockchain[len(Blockchain)-1]
		chainLen := len(Blockchain)
		bcMutex.Unlock()

		if newBlock.PrevHash == myLastBlock.Hash {
			bcMutex.Lock()
			if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
				Blockchain = append(Blockchain, newBlock)
				fmt.Printf("\n--- Appended new block %d ---\n", newBlock.Index)
			}
			bcMutex.Unlock()
		} else if newBlock.Index > chainLen-1 {
			fmt.Printf("Our blockchain (length %d) is behind peer's (length %d). Requesting full chain...\n", chainLen, newBlock.Index+1)
			req := Message{Type: MsgTypeRequestChain, Payload: nil}
			reqBytes, _ := req.Marshal() // Error ignored for simplicity in broadcast
			topic.Publish(ctx, reqBytes)
		}

	case MsgTypeRequestChain:
		bcMutex.Lock()
		serializableChain := make([]SerializableBlock, len(Blockchain))
		for i, b := range Blockchain {
			serializableChain[i], _ = b.toSerializable()
		}
		chainBytes, _ := json.Marshal(serializableChain)
		bcMutex.Unlock()

		resp := Message{Type: MsgTypeRespondChain, Payload: chainBytes}
		respBytes, _ := resp.Marshal()
		topic.Publish(ctx, respBytes)
		fmt.Println("A peer requested our chain. Sent it.")

	case MsgTypeRespondChain:
		var theirChain []SerializableBlock
		if err := json.Unmarshal(message.Payload, &theirChain); err != nil {
			fmt.Println("Error unmarshaling peer's chain response:", err)
			return
		}

		richChain := make([]Block, len(theirChain))
		var err error // Declare err once for the loop
		for i, sb := range theirChain {
			richChain[i], err = sb.toBlock()
			if err != nil {
				fmt.Println("Error converting peer's block:", err)
				return
			}
		}

		bcMutex.Lock()
		if len(richChain) > len(Blockchain) && isChainValid(richChain) {
			Blockchain = richChain
			fmt.Println("Replaced our chain with a longer, valid chain from the network.")
		}
		bcMutex.Unlock()
	}
}

// readFromTopic now passes the 'node' object to the handler.
func readFromTopic(ctx context.Context, node host.Host, topic *pubsub.Topic) {
	sub, err := topic.Subscribe()
	if err != nil {
		fmt.Println("Error subscribing to topic:", err)
		return
	}
	defer sub.Cancel()

	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			// This can happen on shutdown, so we just return
			return
		}
		go handleTopicMessage(ctx, node, topic, msg)
	}
}

// findPeers was missing. Here it is again.
func findPeers(ctx context.Context, node host.Host, discovery *routing.RoutingDiscovery) {
    // This now works because time is imported
	ticker := time.NewTicker(1 * time.Minute) 
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Println("Searching for other De-Book peers...")
            // This now works because DiscoveryTag is defined in main.go
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
					// Muted to avoid spam
				} else {
					fmt.Printf("Connected to a new De-Book peer: %s\n", p.ID.ShortString())
				}
			}
		}
	}
}