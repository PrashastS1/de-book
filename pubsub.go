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
func setupDiscoveryAndPubSub(ctx context.Context, node host.Host, kadDHT *dht.IpfsDHT, printer func(string, ...interface{})) (*pubsub.Topic, error) {
	ps, err := pubsub.NewGossipSub(ctx, node)
	if err != nil { return nil, fmt.Errorf("failed to create pubsub: %w", err) }

	topic, err := ps.Join(DeBookTopic)
	if err != nil { return nil, fmt.Errorf("failed to join topic: %w", err) }

	go readFromTopic(ctx, node, topic, printer)

	routingDiscovery := routing.NewRoutingDiscovery(kadDHT)
	util.Advertise(ctx, routingDiscovery, DiscoveryTag)
	printer("Successfully advertised our presence on the network!")
	go findPeers(ctx, node, routingDiscovery, printer)

	return topic, nil
}

func readFromTopic(ctx context.Context, node host.Host, topic *pubsub.Topic, printer func(string, ...interface{})) {
	sub, err := topic.Subscribe()
	if err != nil { printer("Error subscribing to topic: %v", err); return }
	defer sub.Cancel()

	for {
		msg, err := sub.Next(ctx)
		if err != nil { return } // Exit on shutdown
		go handleTopicMessage(ctx, node, topic, msg, printer)
	}
}


// handleTopicMessage is the new message router.
// We need to pass 'node' into it to get the local peer ID.
func handleTopicMessage(ctx context.Context, node host.Host, topic *pubsub.Topic, msg *pubsub.Message, printer func(string, ...interface{})) {
	if msg.ReceivedFrom == node.ID() { return }

	var message Message
	if err := json.Unmarshal(msg.Data, &message); err != nil { return }

	switch message.Type {
	case MsgTypeAnnounceBlock:
		var sb SerializableBlock
		if err := json.Unmarshal(message.Payload, &sb); err != nil { return }
		newBlock, err := sb.toBlock()
		if err != nil { return }

		bcMutex.Lock()
		myLastBlock := Blockchain[len(Blockchain)-1]
		chainLen := len(Blockchain)
		bcMutex.Unlock()

		if newBlock.PrevHash == myLastBlock.Hash {
			bcMutex.Lock()
			if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
				Blockchain = append(Blockchain, newBlock)
				printer("\n--- Appended new block %d ---", newBlock.Index)
			}
			bcMutex.Unlock()
		} else if newBlock.Index > chainLen-1 {
			printer("Our blockchain (length %d) is behind. Requesting full chain...", chainLen)
			req := Message{Type: MsgTypeRequestChain}
			reqBytes, _ := req.Marshal()
			topic.Publish(ctx, reqBytes)
		}

	case MsgTypeRequestChain:
		bcMutex.Lock()
		serializableChain := make([]SerializableBlock, len(Blockchain))
		for i, b := range Blockchain { serializableChain[i], _ = b.toSerializable() }
		chainBytes, _ := json.Marshal(serializableChain)
		bcMutex.Unlock()
		resp := Message{Type: MsgTypeRespondChain, Payload: chainBytes}
		respBytes, _ := resp.Marshal()
		topic.Publish(ctx, respBytes)
		printer("A peer requested our chain. Sent it.")

	case MsgTypeRespondChain:
		var theirChain []SerializableBlock
		if err := json.Unmarshal(message.Payload, &theirChain); err != nil { return }
		richChain := make([]Block, len(theirChain))
		var err error
		for i, sb := range theirChain {
			richChain[i], err = sb.toBlock()
			if err != nil { return }
		}
		bcMutex.Lock()
		if len(richChain) > len(Blockchain) && isChainValid(richChain) {
			Blockchain = richChain
			printer("Replaced our chain with a longer, valid chain from the network.")
		}
		bcMutex.Unlock()
	}
}

func findPeers(ctx context.Context, node host.Host, discovery *routing.RoutingDiscovery, printer func(string, ...interface{})) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			printer("Searching for other De-Book peers...")
			peerChan, err := discovery.FindPeers(ctx, DiscoveryTag)
			if err != nil {
				printer("Failed to find peers: %v", err)
				continue
			}
			for p := range peerChan {
				if p.ID == node.ID() || len(p.Addrs) == 0 { continue }
				if err := node.Connect(ctx, p); err == nil {
					printer("Connected to a new De-Book peer: %s", p.ID.ShortString())
				}
			}
		}
	}
}