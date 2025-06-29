# De-Book: A Decentralized Peer-to-Peer Book Exchange Platform

![Go Version](https://img.shields.io/badge/Go-1.18%2B-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)

De-Book is a proof-of-concept desktop application that enables users to exchange physical books in a completely decentralized, serverless manner. Instead of relying on a central server, each user's application acts as a node in a peer-to-peer network, discovering other users, sharing data directly, and maintaining a shared, secure ledger of all book ownership and trades.

This project was built to explore and solve real-world distributed systems problems, including P2P networking, data synchronization, consensus, and establishing trust in a trustless environment.

## Demo

![De-Book Demo GIF](https://your-image-hosting-url/de-book-demo.gif)

*To create a GIF like this, you can use a tool like [asciinema](https://asciinema.org/) combined with `asciinema2gif`, or a screen recorder like Kap or Giphy Capture.*

## Core Concepts & Technical Deep Dive

This project goes far beyond a typical CRUD application by tackling several difficult distributed systems challenges.

### 1. Peer-to-Peer (P2P) Networking with `libp2p`

*   **Problem:** How can users find and communicate with each other directly without a central server to broker connections?
*   **Solution:** The application uses `go-libp2p`, the networking stack of IPFS and Ethereum 2.0.
    *   **Peer Discovery:** A Kademlia Distributed Hash Table (DHT) is used to discover other De-Book peers on the network. New nodes bootstrap into the network by connecting to known public peers.
    *   **NAT Traversal:** `libp2p`'s AutoNAT and relay features are enabled, allowing peers behind home routers (NATs) to connect to each other successfully.
    *   **Secure Communication:** All communication between peers is automatically encrypted and authenticated using the keys generated at startup.
*   **Why it's Impressive:** This demonstrates a "truly serverless" architecture. The system is resilient and has no single point of failure, relying on the same robust networking principles as major cryptocurrencies.

### 2. Distributed Ledger & Data Synchronization

*   **Problem:** Without a central database, how do you maintain a consistent and trustworthy record of who owns which book?
*   **Solution:** De-Book implements a purpose-built, lightweight blockchain (an append-only log).
    *   **Transactions:** Every action (`REGISTER_BOOK`, `PROPOSE_TRADE`, `CONFIRM_TRADE`) is a cryptographically signed "block" that contains the transaction data, a timestamp, and the hash of the previous block.
    *   **Data Propagation:** When a user creates a new block, it's broadcast to the network using `libp2p`'s GossipSub (a Pub/Sub protocol). Other peers receive the block, validate it, and append it to their local copy of the chain.

### 3. Consensus via "Longest Chain Wins"

*   **Problem:** What happens if two peers are temporarily disconnected, creating a "fork" in the blockchain history?
*   **Solution:** A simple but effective consensus rule is implemented: "the longest valid chain wins."
    *   When a peer receives a block that doesn't fit its current chain, it recognizes it's out of sync.
    *   It then broadcasts a request for the full chain from its peers.
    *   Upon receiving a longer chain, it validates the *entire* chain from the deterministic Genesis Block onward. If the longer chain is valid, the peer replaces its local chain with the new, longer one.

### 4. Hybrid Decentralized Architecture with IPFS

*   **Problem:** Storing large files like book cover images directly on a blockchain is inefficient and leads to bloat.
*   **Solution:** De-Book integrates with the InterPlanetary File System (IPFS).
    *   When a user lists a book with a cover, the image file is added to their local IPFS node.
    *   IPFS returns a unique, permanent Content Identifier (CID).
    *   Only this lightweight CID string is stored on the De-Book ledger.
    *   The application then displays a public gateway link (`https://ipfs.io/ipfs/<CID>`) for anyone to view the cover.
*   **Why it's Impressive:** This demonstrates an understanding of using the right tool for the job—our ledger for small, critical transaction data, and IPFS for large, immutable file storage.

### 5. On-Chain Reputation System

*   **Problem:** In an anonymous system, how can you know which peers are reliable?
*   **Solution:** The `view` command includes a state processor that analyzes the entire blockchain history to calculate a reputation score for each peer.
    *   **+2 points:** For every successfully completed trade (for both participants).
    *   **-1 point:** For every trade proposal that remains uncompleted after 7 days (to penalize spam).
*   **Why it's Impressive:** This shows the ability to derive valuable, high-level insights from raw, on-chain data, addressing the critical issue of trust in a decentralized exchange.

## Tech Stack

*   **Backend:** Go (Golang)
*   **Networking:** `go-libp2p`
    *   Peer Discovery: Kademlia DHT
    *   Pub/Sub: GossipSub
*   **Storage Integration:** `go-ipfs-api` for communicating with an IPFS daemon.
*   **Data Serialization:** JSON
*   **Identity:** ECDSA Public/Private Key Cryptography

## Setup and Installation

You will need Go and an IPFS node to run this application.

### 1. Prerequisites
*   **Go:** Version 1.18 or higher. [Installation Guide](https://go.dev/doc/install)
*   **IPFS Desktop:** The easiest way to run an IPFS daemon. [Installation Guide](https://ipfs.tech/install/desktop/)

### 2. Installation
1.  **Clone the repository:**
    ```bash
    git clone https://github.com/your-username/de-book.git
    cd de-book
    ```
2.  **Build the application:**
    ```bash
    go build
    ```

### 3. Running the Application
To properly test the P2P functionality, you need to run at least two instances (peers).

1.  **Launch IPFS Desktop** and make sure its status is "Running".

2.  **Run the First Peer (Peer A):**
    *   Open your first terminal window.
    *   Navigate to the project directory.
    *   Run the command:
        ```bash
        ./de-book
        ```
    *   It will create an `identity.key` file and start the node. Note its Peer ID from the startup message.

3.  **Run the Second Peer (Peer B):**
    *   To ensure the second peer has a unique identity, we must run it from a separate directory.
    *   Open a second terminal window.
    *   From the parent directory of your project, copy the project folder:
        ```bash
        # If you are in ~/projects/ and your project is de-book:
        cp -r de-book de-book-peer2
        ```
    *   Navigate into the new directory and remove the old identity file:
        ```bash
        cd de-book-peer2
        rm identity.key
        ```
    *   Run the second peer:
        ```bash
        ./de-book
        ```
    *   Wait a minute, and you should see a "Connected to a new De-Book peer" message in one of the terminals.

## Usage (CLI Commands)

*   `list <ISBN> <Title> <Author> [optional_image_path]`
    *   Registers a book you own to the ledger. If you provide a path to an image, it will be added to IPFS.
    *   *Example:* `list 978-0134190440 EffectiveGo Donovan /path/to/cover.jpg`

*   `view`
    *   Scans the entire blockchain to compute and display the current state of the system, including all available books and their owners, pending trade proposals, and peer reputation scores.

*   `propose <target_peer_id> <their_isbn> <my_isbn>`
    *   Proposes a trade to another user.
    *   *Example:* `propose 12D3Koo... ISBN-B ISBN-A`

*   `accept <proposal_id>`
    *   Accepts a pending trade proposal directed at you.
    *   *Example:* `accept 5a1b...`

*   `help`
    *   Displays a list of all available commands.

## Future Work & Roadmap

*   **GUI Frontend:** Wrap the Go backend in a cross-platform desktop application using [Tauri](https://tauri.app/) for a rich user experience.
*   **Formalize Protocol:** Replace JSON with Protocol Buffers (Protobuf) for more efficient and strongly-typed network communication.
*   **Full-Text Search:** Implement a client-side search index using a library like BleveSearch to allow users to efficiently search for books.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.