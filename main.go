package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
)

const identityPath = "identity.key"

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
	go runCLI(ctx, node, topic)

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
func runCLI(ctx context.Context, node host.Host, topic *pubsub.Topic) {
	reader := bufio.NewReader(os.Stdin)
	// A small delay to allow the network setup to complete before showing the prompt
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

		switch command {
		case "list":
			if len(parts) < 2 {
				fmt.Println("Usage: list <ISBN> <Title> <Author>")
				fmt.Println("Example: list 978-0321765723 TheGoProgrammingLanguage Donovan&Kernighan")
				continue
			}
			bookDetails := strings.SplitN(parts[1], " ", 3)
			if len(bookDetails) < 3 {
				fmt.Println("Usage: list <ISBN> <Title> <Author>")
				fmt.Println("Example: list 978-0321765723 TheGoProgrammingLanguage Donovan&Kernighan")
				continue
			}

			// Create a book record with the node's own ID as the owner
			book := Book{
				Owner:     node.ID(),
				ISBN:      bookDetails[0],
				Title:     bookDetails[1],
				Author:    bookDetails[2],
				Timestamp: time.Now(),
			}

			// Marshal the book data to JSON and publish it on the topic
			bookBytes, err := book.Marshal()
			if err != nil {
				fmt.Println("Error marshaling book:", err)
				continue
			}
			if err := topic.Publish(ctx, bookBytes); err != nil {
				fmt.Println("Error publishing book:", err)
				continue
			}
			fmt.Println("Book published to the network!")

		case "view":
			fmt.Println("--- Local Book Registry ---")
			if len(bookRegistry) == 0 {
				fmt.Println("No books found yet.")
			}
			for _, book := range bookRegistry {
				fmt.Printf("- Title: %s, Author: %s, ISBN: %s (Owner: %s)\n", book.Title, book.Author, book.ISBN, book.Owner.ShortString())
			}
			fmt.Println("-------------------------")

		case "help":
			fmt.Println("Available commands:")
			fmt.Println("  list <ISBN> <Title> <Author> - List a new book for exchange.")
			fmt.Println("  view                       - View all books discovered on the network.")
			fmt.Println("  exit                       - Shut down the application.")

		case "exit":
			fmt.Println("To exit, please press Ctrl+C.")

		default:
			fmt.Println("Unknown command. Type 'help' for a list of commands.")
		}
	}
}