package main

import (
	"context"
	"fmt"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	// "github.com/libp2p/go-libp2p/core/peer" <-- This is now unused, as pointed out by the compiler! Let's remove it.
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
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
	go readFromTopic(ctx, node, topic) // Pass the 'node' object

	// 4. Set up peer discovery
	routingDiscovery := routing.NewRoutingDiscovery(kadDHT)
	util.Advertise(ctx, routingDiscovery, DiscoveryTag)
	fmt.Println("Successfully advertised our presence on the network!")

	// 5. Start a goroutine to find other peers
	go findPeers(ctx, node, routingDiscovery)

	return topic, nil
}

// readFromTopic continuously reads messages from the pubsub topic and processes them.
// --- MODIFICATION HERE ---
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
			return
		}

		// Don't process our own messages
		// --- MODIFICATION HERE ---
		if msg.ReceivedFrom == node.ID() { // Use the passed-in node's ID
			continue
		}

		var book Book
		if err := book.Unmarshal(msg.Data); err != nil {
			fmt.Println("Error unmarshaling book data:", err)
			continue
		}

		registryKey := fmt.Sprintf("%s:%s", book.Owner.String(), book.ISBN)
		bookRegistry[registryKey] = book

		fmt.Printf("\n--- New Book Received ---\n")
		fmt.Printf("From: %s\n", msg.ReceivedFrom.ShortString())
		fmt.Printf("Title: %s, Author: %s, ISBN: %s\n", book.Title, book.Author, book.ISBN)
		fmt.Printf("-------------------------\n> ")
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