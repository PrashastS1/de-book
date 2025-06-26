package main

import (
	"encoding/pem"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/crypto" // This is the main crypto package we need
)

// createPrivateKey creates a new ECDSA private key.
func createPrivateKey() (crypto.PrivKey, error) {
	// For this project, ECDSA is a good choice.
	priv, _, err := crypto.GenerateKeyPair(crypto.ECDSA, -1) // -1 uses default bit size
	if err != nil {
		return nil, err
	}
	return priv, nil
}

// savePrivateKey saves the private key to a file in PEM format.
func savePrivateKey(privKey crypto.PrivKey, path string) error {
	bytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return err
	}
	pemBlock := &pem.Block{
		Type:  "LIBP2P PRIVATE KEY", // Using a more specific type header
		Bytes: bytes,
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return pem.Encode(file, pemBlock)
}

// loadPrivateKey loads a private key from a PEM file.
// THIS FUNCTION IS CORRECTED.
func loadPrivateKey(path string) (crypto.PrivKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}
	if block.Type != "LIBP2P PRIVATE KEY" {
		return nil, fmt.Errorf("invalid PEM block type: %s", block.Type)
	}
	privKey, err := crypto.UnmarshalPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

// THIS FUNCTION IS NO LONGER NEEDED with the simpler save/load logic.
// The old x509 logic was overly complex. We can remove it.
// The new savePrivateKey and loadPrivateKey work directly with libp2p's marshaling.