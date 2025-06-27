package main

import (
	"bufio"
	"context"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"crypto/rand"
	"syscall"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)
const DeBookTopic = "/de-book/1.0.0"
const DiscoveryTag = "de-book-discovery"
const identityPath = "identity.key"

func generateUUID() string {
    bytes := make([]byte, 16)
    rand.Read(bytes)
    return hex.EncodeToString(bytes)
}

func main() {
	// --- Identity Loading ---
	var privKey crypto.PrivKey
	var err error

	if _, err := os.Stat(identityPath); os.IsNotExist(err) {
		fmt.Println("No existing identity found. Creating a new one...")
		privKey, err = createPrivateKey()
		if err != nil {
			panic(err)
		}
		if err = savePrivateKey(privKey, identityPath); err != nil {
			panic(err)
		}
		fmt.Println("New identity created and saved to", identityPath)
	} else {
		fmt.Println("Loading existing identity from", identityPath)
		privKey, err = loadPrivateKey(identityPath)
		if err != nil {
			panic(err)
		}
	}

	// --- Node Creation ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node, kadDHT, err := createNode(ctx, privKey)
	if err != nil {
		panic(err)
	}

	createGenesisBlock()
	fmt.Println("Blockchain initialized with Genesis Block.")

	fmt.Println("---------------------------------")
	fmt.Println("Node is running! Your Peer ID is:", node.ID())
	fmt.Println("Your node is listening on addresses:")
	for _, addr := range node.Addrs() {
		fmt.Printf("- %s/p2p/%s\n", addr, node.ID())
	}
	fmt.Println("---------------------------------")

	// --- Setup Discovery and PubSub ---
	topic, err := setupDiscoveryAndPubSub(ctx, node, kadDHT)
	if err != nil {
		panic(err)
	}

	// --- Run the CLI ---
	go runCLI(ctx, node, topic, privKey)

	// --- Wait for a termination signal ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\nReceived shutdown signal, closing node...")
	if err := node.Close(); err != nil {
		panic(err)
	}
	fmt.Println("Node closed.")
}

// // runCLI provides a simple command-line interface for the user.
// func runCLI(ctx context.Context, node host.Host, topic *pubsub.Topic, privKey crypto.PrivKey) {
// 	reader := bufio.NewReader(os.Stdin)
// 	time.Sleep(2 * time.Second)

// 	for {
// 		fmt.Print("> ")
// 		input, err := reader.ReadString('\n')
// 		if err != nil {
// 			fmt.Println("Error reading from stdin:", err)
// 			return
// 		}
// 		input = strings.TrimSpace(input)
// 		if input == "" {
// 			continue
// 		}

// 		parts := strings.SplitN(input, " ", 2)
// 		command := parts[0]

// 		switch command {

// 			case "accept":
// 				// Usage: accept <proposal_id>
// 				if len(parts) < 2 { /* ... usage message ... */ continue }
// 				proposalID := parts[1]

// 				// Here, we would first scan the blockchain to ensure the proposal is valid
// 				// and that we are the intended target. (This is an important validation step!)

// 				tx := Transaction{Type: "CONFIRM_TRADE", ProposalID: proposalID}

// 				// Create and broadcast the confirmation block
// 				newBlock, err := generateBlock(privKey, tx)
// 				if err != nil { /* ... */ }
// 				// ... add to local chain and publish ...
// 				fmt.Println("Trade confirmation published for proposal:", proposalID)

// 			case "propose":
// 				// Usage: propose <target_peer_id> <their_isbn> <my_isbn>
// 				if len(parts) < 2 { /* ... usage message ... */ continue }
// 				tradeDetails := strings.SplitN(parts[1], " ", 3)
// 				if len(tradeDetails) < 3 { /* ... usage message ... */ continue }

// 				proposal := TradeProposal{
// 					ProposerID:  node.ID().String(),
// 					TargetID:    tradeDetails[0],
// 					TargetBookISBN:   tradeDetails[1],
// 					ProposerBookISBN: tradeDetails[2],
// 					ProposalID:  generateUUID(),
// 				}

// 				tx := Transaction{Type: "PROPOSE_TRADE", Trade: proposal}
				
// 				// Create and broadcast the block containing the proposal
// 				// (This logic is the same as the 'list' command)
// 				newBlock, err := generateBlock(privKey, tx)
// 				if err != nil { /* ... */ }
// 				// ... add to local chain and publish ...
// 				fmt.Println("Trade proposal published to the network. Proposal ID:", proposal.ProposalID)

// 			case "list":
// 				if len(parts) < 2 {
// 					fmt.Println("Usage: list <ISBN> <Title> <Author>")
// 					fmt.Println("Example: list 978-0321765723 TheGoProgrammingLanguage Donovan&Kernighan")
// 					continue
// 				}
// 				bookDetails := strings.SplitN(parts[1], " ", 3)
// 				if len(bookDetails) < 3 {
// 					fmt.Println("Usage: list <ISBN> <Title> <Author>")
// 					fmt.Println("Example: list 978-0321765723 TheGoProgrammingLanguage Donovan&Kernighan")
// 					continue
// 				}

// 				// Create a book record with the node's own ID as the owner
// 				// book := Book{
// 				// 	Owner:     node.ID(),
// 				// 	ISBN:      bookDetails[0],
// 				// 	Title:     bookDetails[1],
// 				// 	Author:    bookDetails[2],
// 				// 	Timestamp: time.Now(),
// 				// }

// 				transaction := Transaction{
// 					Type: "REGISTER_BOOK",
// 					Book: Book{
// 						ISBN:   bookDetails[0],
// 						Title:  bookDetails[1],
// 						Author: bookDetails[2],
// 					},
// 				}

// 				newBlock, err := generateBlock(privKey, transaction)
// 				if err != nil {
// 					fmt.Println("Error generating block:", err)
// 					continue
// 				}

// 				bcMutex.Lock()

// 				if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
// 					Blockchain = append(Blockchain, newBlock)
// 				} else {
// 					fmt.Println("Generated an invalid block. Something is wrong.")
// 					bcMutex.Unlock()
// 					continue
// 				}
// 				bcMutex.Unlock()
// 				fmt.Println("New block added to our local ledger.")

// 				serializable, err := newBlock.toSerializable()
// 				if err != nil {
// 					fmt.Println("Error converting block for serialization:", err)
// 					continue
// 				}

// 				payloadBytes, err := json.Marshal(serializable)
// 				if err != nil {
// 					fmt.Println("Error marshaling payload:", err)
// 					continue
// 				}

// 				msg := Message{
// 					Type:    MsgTypeAnnounceBlock,
// 					Payload: payloadBytes,
// 				}

// 				msgBytes, err := msg.Marshal()

// 				if err != nil {
// 					fmt.Println("Error marshaling final message:", err)
// 					continue
// 				}

// 				if err := topic.Publish(ctx, msgBytes); err != nil {
// 					fmt.Println("Error publishing message:", err)
// 					continue
// 				}
// 				fmt.Println("Block announcement published to the network!")

// 				// // 4. Publish the block to the network
// 				// blockBytes, err := json.Marshal(serializable)
// 				// if err != nil {
// 				// 	fmt.Println("Error marshaling block:", err)
// 				// 	continue
// 				// }
// 				// if err := topic.Publish(ctx, blockBytes); err != nil {
// 				// 	fmt.Println("Error publishing block:", err)
// 				// 	continue
// 				// }
// 				// fmt.Println("Block published to the network!")

// 				// // // Marshal the book data to JSON and publish it on the topic
// 				// // bookBytes, err := book.Marshal()
// 				// // if err != nil {
// 				// // 	fmt.Println("Error marshaling book:", err)
// 				// // 	continue
// 				// // }
// 				// // if err := topic.Publish(ctx, bookBytes); err != nil {
// 				// // 	fmt.Println("Error publishing book:", err)
// 				// // 	continue
// 				// // }
// 				// // fmt.Println("Book published to the network!")

// 			// case "view":
// 			// 	fmt.Println("--- Local Book Registry ---")
// 			// 	if len(bookRegistry) == 0 {
// 			// 		fmt.Println("No books found yet.")
// 			// 	}
// 			// 	for _, book := range bookRegistry {
// 			// 		fmt.Printf("- Title: %s, Author: %s, ISBN: %s (Owner: %s)\n", book.Title, book.Author, book.ISBN, book.Owner.ShortString())
// 			// 	}
// 			// 	fmt.Println("-------------------------")

// 			case "view":
// 				fmt.Println("--- Ledger State ---")

// 				bcMutex.Lock()
// 				// Create a deep copy to avoid holding the lock during processing
// 				chainCopy := make([]Block, len(Blockchain))
// 				copy(chainCopy, Blockchain)
// 				bcMutex.Unlock()

// 				// --- State Processing Maps ---
// 				// bookRegistry stores the static details of a book.
// 				bookRegistry := make(map[string]Book) // Key: ISBN
// 				// bookOwnership stores the current owner of a book.
// 				bookOwnership := make(map[string]peer.ID) // Key: ISBN
// 				// pendingProposals stores trades that have been proposed but not yet confirmed.
// 				pendingProposals := make(map[string]TradeProposal) // Key: ProposalID
// 				// confirmedProposals keeps track of trades that are done.
// 				confirmedProposals := make(map[string]bool) // Key: ProposalID

// 				// --- Replay the Entire Blockchain History ---
// 				for _, block := range chainCopy {
// 					if block.Index == 0 { continue } // Skip Genesis Block

// 					var tx Transaction
// 					if err := json.Unmarshal([]byte(block.Data), &tx); err != nil {
// 						continue // Skip malformed blocks
// 					}

// 					creatorID, err := peer.IDFromPublicKey(block.CreatorPubKey)
// 					if err != nil {
// 						continue
// 					}

// 					switch tx.Type {
// 					case "REGISTER_BOOK":
// 						// A new book is registered to its first owner.
// 						bookRegistry[tx.Book.ISBN] = tx.Book
// 						bookOwnership[tx.Book.ISBN] = creatorID
					
// 					case "PROPOSE_TRADE":
// 						// A trade proposal is made. Store it for later confirmation.
// 						// We could add validation here (e.g., does the proposer own the book?).
// 						// For now, we'll validate at confirmation time.
// 						pendingProposals[tx.Trade.ProposalID] = tx.Trade

// 					case "CONFIRM_TRADE":
// 						// A trade is confirmed. Let's validate and execute it.
// 						proposal, exists := pendingProposals[tx.Trade.ProposalID]
// 						if !exists {
// 							continue // Proposal not found or already completed.
// 						}

// 						// 1. Validate: Is the peer confirming the trade the one it was offered to?
// 						if creatorID.String() != proposal.TargetID {
// 							continue // Wrong person confirmed the trade.
// 						}
						
// 						// 2. Validate: Do both parties currently own the books they are offering?
// 						proposerOwnsBook := bookOwnership[proposal.ProposerBookISBN].String() == proposal.ProposerID
// 						targetOwnsBook := bookOwnership[proposal.TargetBookISBN].String() == proposal.TargetID
						
// 						if !proposerOwnsBook || !targetOwnsBook {
// 							continue // One of the parties no longer owns the book. Trade is void.
// 						}
						
// 						// 3. Execute Swap: All checks passed. Swap ownership.
// 						// The proposer gets the target's book.
// 						// bookOwnership[proposal.TargetBookISBN] = creatorID.String() // string to peer id
						
// 						// The target gets the proposer's book.
// 						proposerPeerID, _ := peer.Decode(proposal.ProposerID)
// 						targetPeerID, _ := peer.Decode(proposal.TargetID)
// 						// bookOwnership[proposal.ProposerBookISBN] = proposerPeerID
// 						bookOwnership[proposal.TargetBookISBN] = proposerPeerID
// 						bookOwnership[proposal.ProposerBookISBN] = targetPeerID


// 						// 4. Clean up: Mark proposal as completed.
// 						delete(pendingProposals, tx.Trade.ProposalID)
// 						confirmedProposals[tx.Trade.ProposalID] = true
// 					}
// 				}

// 				// --- Print the Final State ---
// 				fmt.Println("\n[Available Books]")
// 				if len(bookOwnership) == 0 {
// 					fmt.Println("No books registered on the ledger.")
// 				}
// 				for isbn, ownerID := range bookOwnership {
// 					book := bookRegistry[isbn]
// 					fmt.Printf("- Title: %-30s | Author: %-20s | ISBN: %-20s | Owner: %s\n", 
// 						book.Title, book.Author, book.ISBN, ownerID.ShortString())
// 				}
				
// 				fmt.Println("\n[Pending Trade Proposals]")
// 				if len(pendingProposals) == 0 {
// 					fmt.Println("No pending proposals.")
// 				}
// 				for id, proposal := range pendingProposals {
// 					fmt.Printf("- ID: %s | Offer: %s's book (%s) for %s's book (%s)\n",
// 						id,
// 						peer.ID(proposal.ProposerID).ShortString(),
// 						proposal.ProposerBookISBN,
// 						peer.ID(proposal.TargetID).ShortString(),
// 						proposal.TargetBookISBN)
// 				}
// 				fmt.Println("--------------------")

// 			// case "view":
// 			// 	fmt.Println("--- Current State from Ledger ---")
// 			// 	// This is a simple state reconstruction. A real app would cache this.
// 			// 	bcMutex.Lock()

// 			// 	chainCopy := make([]Block, len(Blockchain))
// 			// 	copy(chainCopy, Blockchain)
				
// 			// 	bcMutex.Unlock()

// 			// 	bookOwnership := make(map[string]peer.ID) // Key: ISBN, Value: Owner Peer ID

// 			// 	// Iterate over the blockchain to find the latest owner of each book
// 			// 	for _, block := range chainCopy {
// 			// 		if block.Index == 0 { continue } // Skip Genesis

// 			// 		var tx Transaction
// 			// 		if err := json.Unmarshal([]byte(block.Data), &tx); err != nil {
// 			// 			continue // Skip malformed blocks
// 			// 		}

// 			// 		if tx.Type == "REGISTER_BOOK" {
// 			// 			ownerID, _ := peer.IDFromPublicKey(block.CreatorPubKey)
// 			// 			bookOwnership[tx.Book.ISBN] = ownerID
// 			// 		}
// 			// 		// Later, we would add "TRADE_BOOK" logic here to update the owner
// 			// 	}

// 			// 	if len(bookOwnership) == 0 {
// 			// 		fmt.Println("No books registered on the ledger.")
// 			// 	}
// 			// 	for isbn, owner := range bookOwnership {
// 			// 		// We need to find the book details again for printing
// 			// 		var title, author string
// 			// 		for _, block := range Blockchain {
// 			// 			var tx Transaction
// 			// 			if err := json.Unmarshal([]byte(block.Data), &tx); err == nil && tx.Book.ISBN == isbn {
// 			// 				title = tx.Book.Title
// 			// 				author = tx.Book.Author
// 			// 				break
// 			// 			}
// 			// 		}
// 			// 		fmt.Printf("- Title: %s, Author: %s, ISBN: %s (Owner: %s)\n", title, author, isbn, owner.ShortString())
// 			// 	}
// 			// 	fmt.Println("---------------------------------")

// 			case "help":
// 				fmt.Println("Available commands:")
// 				fmt.Println("  list <ISBN> <Title> <Author> - List a new book for exchange.")
// 				fmt.Println("  view                       - View all books discovered on the network.")
// 				fmt.Println("  exit                       - Shut down the application.")

// 			case "exit":
// 				fmt.Println("To exit, please press Ctrl+C.")

// 			default:
// 				fmt.Println("Unknown command. Type 'help' for a list of commands.")
// 			}
// 	}
// }

// runCLI provides a simple command-line interface for the user.
func runCLI(ctx context.Context, node host.Host, topic *pubsub.Topic, privKey crypto.PrivKey) {
	reader := bufio.NewReader(os.Stdin)
	time.Sleep(2 * time.Second)

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin:", err)
			return
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		parts := strings.SplitN(input, " ", 2)
		command := parts[0]

		// This helper function reduces code duplication for creating and broadcasting a block.
		publishBlock := func(tx Transaction) {
			newBlock, err := generateBlock(privKey, tx)
			if err != nil {
				fmt.Println("Error generating block:", err)
				return
			}

			bcMutex.Lock()
			if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
				Blockchain = append(Blockchain, newBlock)
			} else {
				fmt.Println("Generated an invalid block. Something is wrong.")
				bcMutex.Unlock()
				return
			}
			bcMutex.Unlock()
			fmt.Println("New block added to our local ledger.")

			serializable, err := newBlock.toSerializable()
			if err != nil {
				fmt.Println("Error converting block for serialization:", err)
				return
			}
			payloadBytes, err := json.Marshal(serializable)
			if err != nil {
				fmt.Println("Error marshaling payload:", err)
				return
			}
			msg := Message{Type: MsgTypeAnnounceBlock, Payload: payloadBytes}
			msgBytes, err := msg.Marshal()
			if err != nil {
				fmt.Println("Error marshaling final message:", err)
				return
			}
			if err := topic.Publish(ctx, msgBytes); err != nil {
				fmt.Println("Error publishing message:", err)
				return
			}
			fmt.Println("Block announcement published to the network!")
		}

		switch command {
		case "list":
			if len(parts) < 2 {
				fmt.Println("Usage: list <ISBN> <Title> <Author>")
				continue
			}
			bookDetails := strings.SplitN(parts[1], " ", 3)
			if len(bookDetails) < 3 {
				fmt.Println("Usage: list <ISBN> <Title> <Author>")
				continue
			}

			transaction := Transaction{
				Type: "REGISTER_BOOK",
				Book: Book{
					ISBN:   bookDetails[0],
					Title:  bookDetails[1],
					Author: bookDetails[2],
				},
			}
			publishBlock(transaction)

		case "propose":
			if len(parts) < 2 {
				fmt.Println("Usage: propose <target_peer_id> <their_isbn> <my_isbn>")
				continue
			}
			tradeDetails := strings.SplitN(parts[1], " ", 3)
			if len(tradeDetails) < 3 {
				fmt.Println("Usage: propose <target_peer_id> <their_isbn> <my_isbn>")
				continue
			}

			proposal := TradeProposal{
				ProposerID:       node.ID().String(),
				TargetID:         tradeDetails[0],
				TargetBookISBN:   tradeDetails[1],
				ProposerBookISBN: tradeDetails[2],
				ProposalID:       generateUUID(),
			}
			transaction := Transaction{Type: "PROPOSE_TRADE", Trade: proposal}
			publishBlock(transaction)
			fmt.Println("Trade proposal published. Proposal ID:", proposal.ProposalID)

		case "accept":
			if len(parts) < 2 {
				fmt.Println("Usage: accept <proposal_id>")
				continue
			}
			proposalID := parts[1]
			// In a real app, you'd scan the chain here to ensure the proposal is valid before accepting.
			// For now, we trust the user.
			transaction := Transaction{Type: "CONFIRM_TRADE", ProposalID: proposalID}
			publishBlock(transaction)
			fmt.Println("Trade confirmation published for proposal:", proposalID)

		case "view":
			fmt.Println("--- Ledger State ---")

			bcMutex.Lock()
			chainCopy := make([]Block, len(Blockchain))
			copy(chainCopy, Blockchain)
			bcMutex.Unlock()

			bookRegistry := make(map[string]Book)
			bookOwnership := make(map[string]peer.ID)
			pendingProposals := make(map[string]TradeProposal)

			for _, block := range chainCopy {
				if block.Index == 0 { continue }

				var tx Transaction
				if err := json.Unmarshal([]byte(block.Data), &tx); err != nil { continue }
				if block.CreatorPubKey == nil { continue } // Should not happen post-genesis, but a good guard.
				
				creatorID, err := peer.IDFromPublicKey(block.CreatorPubKey)
				if err != nil { continue }

				switch tx.Type {
				case "REGISTER_BOOK":
					bookRegistry[tx.Book.ISBN] = tx.Book
					bookOwnership[tx.Book.ISBN] = creatorID
				
				case "PROPOSE_TRADE":
					pendingProposals[tx.Trade.ProposalID] = tx.Trade

				case "CONFIRM_TRADE":
					proposal, exists := pendingProposals[tx.Trade.ProposalID]
					if !exists { continue }

					if creatorID.String() != proposal.TargetID { continue }
					
					proposerOwnsBook := bookOwnership[proposal.ProposerBookISBN].String() == proposal.ProposerID
					targetOwnsBook := bookOwnership[proposal.TargetBookISBN].String() == proposal.TargetID
					if !proposerOwnsBook || !targetOwnsBook { continue }
					
					proposerPeerID, _ := peer.Decode(proposal.ProposerID)
					targetPeerID, _ := peer.Decode(proposal.TargetID)
					
					bookOwnership[proposal.TargetBookISBN] = proposerPeerID
					bookOwnership[proposal.ProposerBookISBN] = targetPeerID

					delete(pendingProposals, tx.Trade.ProposalID)
				}
			}

			fmt.Println("\n[Available Books]")
			if len(bookOwnership) == 0 {
				fmt.Println("No books registered on the ledger.")
			}
			for isbn, ownerID := range bookOwnership {
				book := bookRegistry[isbn]
				fmt.Printf("- Title: %-30s | Author: %-20s | ISBN: %-20s | Owner: %s\n",
					book.Title, book.Author, book.ISBN, ownerID.ShortString())
			}

			fmt.Println("\n[Pending Trade Proposals]")
			if len(pendingProposals) == 0 {
				fmt.Println("No pending proposals.")
			}
			for id, proposal := range pendingProposals {
                // --- THE FIX IS HERE ---
                // Safely decode the peer ID strings before trying to use them.
				pID, _ := peer.Decode(proposal.ProposerID)
				tID, _ := peer.Decode(proposal.TargetID)
				fmt.Printf("- ID: %s | Offer: %s's book (%s) for %s's book (%s)\n",
					id,
					pID.ShortString(),
					proposal.ProposerBookISBN,
					tID.ShortString(),
					proposal.TargetBookISBN)
			}
			fmt.Println("--------------------")
		
		case "help":
			fmt.Println("Available commands:")
			fmt.Println("  list <ISBN> <Title> <Author>   - Register a book you own.")
			fmt.Println("  propose <PeerID> <TheirISBN> <MyISBN> - Propose a trade.")
			fmt.Println("  accept <ProposalID>            - Accept a proposed trade.")
			fmt.Println("  view                         - View the current state of all books and trades.")
			fmt.Println("  exit                         - Shut down the application.")

		case "exit":
			fmt.Println("To exit, please press Ctrl+C.")

		default:
			fmt.Println("Unknown command. Type 'help' for a list of commands.")
		}
	}
}