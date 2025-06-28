package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
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
var printMutex = &sync.Mutex{}

func safePrint(format string, args ...interface{}) {
	printMutex.Lock()
	defer printMutex.Unlock()
	// Erase the current line, print the message, and redraw the prompt.
	fmt.Printf("\r\033[K")
	fmt.Printf(format+"\n", args...)
	fmt.Print("> ")
}

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
	// topic, err := setupDiscoveryAndPubSub(ctx, node, kadDHT)
	// topic, err := setupDiscoveryAndPubSub(ctx, node, kadDHT, safePrint)
	topic, err := setupDiscoveryAndPubSub(ctx, node, kadDHT, safePrint)
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

// runCLI provides a simple command-line interface for the user.
func runCLI(ctx context.Context, node host.Host, topic *pubsub.Topic, privKey crypto.PrivKey) {
	reader := bufio.NewReader(os.Stdin)
	time.Sleep(1 * time.Second) // Shorter sleep
	fmt.Print("> ")

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			safePrint("Error reading from stdin: %v", err)
			return
		}
		// input = strings.TrimSpace(input)
		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Print("> ")
			continue
		}

		// parts := strings.SplitN(input, " ", 2)
		parts := strings.Fields(input)
		command := parts[0]

		// This helper function reduces code duplication for creating and broadcasting a block.
		publishBlock := func(tx Transaction) {
			newBlock, err := generateBlock(privKey, tx)
			if err != nil {
				safePrint("Error generating block: %v", err)
				return
			}

			bcMutex.Lock()
			if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
				Blockchain = append(Blockchain, newBlock)
			} else {
				safePrint("Generated an invalid block. Something is wrong.")
				bcMutex.Unlock()
				return
			}
			bcMutex.Unlock()
			// fmt.Println("New block added to our local ledger.")
			safePrint("New block added to our local ledger.")

			serializable, err := newBlock.toSerializable()
			if err != nil {
				safePrint("Error converting block for serialization: %v", err)
				return
			}
			payloadBytes, err := json.Marshal(serializable)
			if err != nil {
				safePrint("Error marshaling payload: %v", err)
				return
			}
			msg := Message{Type: MsgTypeAnnounceBlock, Payload: payloadBytes}
			msgBytes, err := msg.Marshal()
			if err != nil {
				safePrint("Error marshaling final message: %v", err)
				return
			}
			if err := topic.Publish(ctx, msgBytes); err != nil {
				safePrint("Error publishing message: %v", err)
				return
			}
			safePrint("Block announcement published to the network!")
		}

		switch command {

			case "list":
				if len(parts) != 4 {
					safePrint("Usage: list <ISBN> <Title> <Author>")
				} else {
					transaction := Transaction{Type: "REGISTER_BOOK", Book: Book{ISBN: parts[1], Title: parts[2], Author: parts[3]}}
					publishBlock(transaction)
				}

			case "propose":
				if len(parts) != 4 {
					safePrint("Usage: propose <target_peer_id> <their_isbn> <my_isbn>")
				} else {
					proposal := TradeProposal{
						ProposerID: node.ID().String(), TargetID: parts[1],
						TargetBookISBN: parts[2], ProposerBookISBN: parts[3],
						ProposalID: generateUUID(),
					}
					transaction := Transaction{Type: "PROPOSE_TRADE", Trade: proposal}
					publishBlock(transaction)
					safePrint("Trade proposal published. Proposal ID: %s", proposal.ProposalID)
				}

			case "accept":
				
				if len(parts) != 2 {
					safePrint("Usage: accept <proposal_id>")
				} else {
					transaction := Transaction{Type: "CONFIRM_TRADE", ProposalID: parts[1]}
					publishBlock(transaction)
					safePrint("Trade confirmation published for proposal: %s", parts[1])
				}

			case "view":

				var b strings.Builder
				b.WriteString("\n--- Ledger State ---\n")

				bcMutex.Lock()
				chainCopy := make([]Block, len(Blockchain))
				copy(chainCopy, Blockchain)
				bcMutex.Unlock()

				bookRegistry := make(map[string]Book)
				bookOwnership := make(map[string]peer.ID)
				pendingProposals := make(map[string]TradeProposal)

				reputation := make(map[peer.ID]int)
				// --- NEW: Define the timeout for stale proposals ---
				const staleProposalTimeout = 7 * 24 * time.Hour // 7 days


				for _, block := range chainCopy {
					if block.Index == 0 { continue }
					var tx Transaction
					if err := json.Unmarshal([]byte(block.Data), &tx); err != nil { continue }
					if block.CreatorPubKey == nil { continue }
					creatorID, err := peer.IDFromPublicKey(block.CreatorPubKey)
					if err != nil { continue }

					if _, exists := reputation[creatorID]; !exists {
						reputation[creatorID] = 0 // Start everyone at 0
					}

					switch tx.Type {
					case "REGISTER_BOOK":
						bookRegistry[tx.Book.ISBN] = tx.Book
						bookOwnership[tx.Book.ISBN] = creatorID
					case "PROPOSE_TRADE":
						proposerID, _ := peer.Decode(tx.Trade.ProposerID)
						targetID, _ := peer.Decode(tx.Trade.TargetID)
						if _, exists := reputation[proposerID]; !exists {
							reputation[proposerID] = 0
						}
						if _, exists := reputation[targetID]; !exists {
							reputation[targetID] = 0
						}
						pendingProposals[tx.Trade.ProposalID] = tx.Trade
					case "CONFIRM_TRADE":

						proposal, exists := pendingProposals[tx.ProposalID]
						if !exists { continue }

						if creatorID.String() != proposal.TargetID { continue }
						
						proposerOwnsBook := bookOwnership[proposal.ProposerBookISBN].String() == proposal.ProposerID
						targetOwnsBook := bookOwnership[proposal.TargetBookISBN].String() == proposal.TargetID
						if !proposerOwnsBook || !targetOwnsBook { continue }
						
						proposerPeerID, _ := peer.Decode(proposal.ProposerID)
						targetPeerID, _ := peer.Decode(proposal.TargetID)
						
						// --- NEW: Update reputation on successful trade ---
						reputation[proposerPeerID] += 2
						reputation[targetPeerID] += 2
						
						bookOwnership[proposal.TargetBookISBN] = proposerPeerID
						bookOwnership[proposal.ProposerBookISBN] = targetPeerID

						delete(pendingProposals, tx.ProposalID)
					}
				}

				for _, proposal := range pendingProposals {
					// We need the original block's timestamp to check if it's stale.
					// This requires a slightly more complex lookup. Let's find the proposal block.
					var proposalBlockTimestamp time.Time
					for _, block := range chainCopy {
						var tx Transaction
						_ = json.Unmarshal([]byte(block.Data), &tx)
						if tx.Type == "PROPOSE_TRADE" && tx.Trade.ProposalID == proposal.ProposalID {
							proposalBlockTimestamp, _ = time.Parse(time.RFC3339, block.Timestamp)
							break
						}
					}
	
					if !proposalBlockTimestamp.IsZero() && time.Since(proposalBlockTimestamp) > staleProposalTimeout {
						proposerID, _ := peer.Decode(proposal.ProposerID)
						reputation[proposerID] -= 1
					}
				}

				b.WriteString("\n[Available Books]\n")
				
				if len(bookOwnership) == 0 { b.WriteString("No books registered on the ledger.\n") }
				for isbn, ownerID := range bookOwnership {
					book := bookRegistry[isbn]
					b.WriteString(fmt.Sprintf("- Title: %-30s | Author: %-20s | ISBN: %-20s | Owner: %s\n",
						book.Title, book.Author, book.ISBN, ownerID.ShortString()))
				}

				b.WriteString("\n[Pending Trade Proposals]\n")
				if len(pendingProposals) == 0 { b.WriteString("No pending proposals.\n") }
				for id, proposal := range pendingProposals {
					pID, _ := peer.Decode(proposal.ProposerID)
					tID, _ := peer.Decode(proposal.TargetID)
					b.WriteString(fmt.Sprintf("- ID: %s | Offer: %s's book (%s) for %s's book (%s)\n",
						id, pID.ShortString(), proposal.ProposerBookISBN, tID.ShortString(), proposal.TargetBookISBN))
				}

				b.WriteString("\n[Peer Reputations]\n")
				if len(reputation) == 0 {
					b.WriteString("No peer activity recorded yet.\n")
				}
				for peerID, score := range reputation {
					b.WriteString(fmt.Sprintf("- Peer: %-20s | Score: %d\n", peerID.ShortString(), score))
				}
				
				b.WriteString("--------------------")
				safePrint(b.String())
			
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
		fmt.Print("> ")
	}
}