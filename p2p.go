package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	// THE FIX IS HERE:
	"github.com/libp2p/go-libp2p/core/crypto" 

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// A list of bootstrap peers provided by the IPFS team.
// These are used to connect to the wider IPFS network.
var bootstrapPeers = []string{
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gXikGcNFwhrt",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTf5sheLSdQem3evXW5",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	"/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
}

// createNode creates a new libp2p Host
// The type 'crypto.PrivKey' is now correctly defined because of the fixed import.
func createNode(ctx context.Context, privKey crypto.PrivKey) (host.Host, *dht.IpfsDHT, error) {
	// 0.0.0.0 will listen on all available network interfaces
	listenAddr := libp2p.ListenAddrStrings(
		"/ip4/0.0.0.0/tcp/0", // Listen on any available TCP port
		"/ip4/0.0.0.0/udp/0/quic", // Listen on any available QUIC port
	)

	// Create a new libp2p host
	node, err := libp2p.New(
		libp2p.Identity(privKey),
		listenAddr,
		// Enable NAT traversal using AutoNAT
		libp2p.EnableNATService(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		return nil, nil, err
	}

	// Create a new DHT
	kadDHT, err := dht.New(ctx, node)
	if err != nil {
		return nil, nil, err
	}
	
	// In server mode, the DHT will answer queries from other peers
	if err = kadDHT.Bootstrap(ctx); err != nil {
		return nil, nil, err
	}

	// Connect to bootstrap peers
	connectToBootstrapPeers(ctx, node)

	return node, kadDHT, nil
}


func connectToBootstrapPeers(ctx context.Context, node host.Host) {
	var wg sync.WaitGroup
	for _, addrStr := range bootstrapPeers {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			fmt.Println("Error parsing bootstrap peer address:", err)
			continue
		}
		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			fmt.Println("Error getting peer info from address:", err)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			// We use a separate context with a timeout for each connection attempt
			ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			if err := node.Connect(ctx, *peerInfo); err != nil {
				// Don't panic, just log the error. It's okay if we can't connect to all of them.
				// fmt.Printf("Failed to connect to bootstrap peer %s: %s\n", peerInfo.ID, err)
			} else {
				fmt.Println("Connection established with bootstrap peer:", peerInfo.ID)
			}
		}()
	}
	wg.Wait()
}