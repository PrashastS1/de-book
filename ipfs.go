package main

import (
	"fmt"
	"os"

	shell "github.com/ipfs/go-ipfs-api"
)

// The default API address for a local IPFS daemon.
const ipfsApiUrl = "/ip4/127.0.0.1/tcp/5001"

// addFileToIPFS takes a file path, adds the file to the local IPFS node,
// and returns the resulting Content ID (CID).
func addFileToIPFS(filePath string) (string, error) {
	// Create a new IPFS shell object
	sh := shell.NewShell(ipfsApiUrl)

	// Check if the daemon is running
	if !sh.IsUp() {
		return "", fmt.Errorf("IPFS daemon is not running")
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	// Add the file to IPFS
	// The second argument to Add can be an io.Reader
	cid, err := sh.Add(file)
	if err != nil {
		return "", fmt.Errorf("failed to add file to IPFS: %w", err)
	}

	fmt.Printf("File added to IPFS with CID: %s\n", cid)
	return cid, nil
}