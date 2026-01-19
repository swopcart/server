package session

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/swopcart/server/internal/config"
)

func TestLoadOrCreateKeyPair_CreatesNewKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	cfg := &config.Config{
		Auth: config.Auth{
			KeyPath: keyPath,
		},
	}

	kp, err := loadOrCreateKeyPair(cfg)
	if err != nil {
		t.Fatalf("loadOrCreateKeyPair failed: %v", err)
	}

	if kp.private == nil {
		t.Error("expected private key to be set")
	}

	if kp.public == nil {
		t.Error("expected public key to be set")
	}

	// Verify the key file was created
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("expected key file to be created")
	}

	// Verify the key file contains valid PEM data
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read key file: %v", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("failed to decode PEM block")
	}

	if block.Type != "RSA PRIVATE KEY" {
		t.Errorf("expected PEM type %q, got %q", "RSA PRIVATE KEY", block.Type)
	}
}

func TestLoadOrCreateKeyPair_LoadsExistingKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "existing.key")

	// Generate and save a key first
	originalKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	keyFile, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}

	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(originalKey),
	}

	if err := pem.Encode(keyFile, pemBlock); err != nil {
		t.Fatalf("failed to encode key: %v", err)
	}
	_ = keyFile.Close()

	cfg := &config.Config{
		Auth: config.Auth{
			KeyPath: keyPath,
		},
	}

	kp, err := loadOrCreateKeyPair(cfg)
	if err != nil {
		t.Fatalf("loadOrCreateKeyPair failed: %v", err)
	}

	// Verify the loaded key matches the original
	if kp.private.N.Cmp(originalKey.N) != 0 {
		t.Error("loaded key does not match original key")
	}
}

func TestLoadOrCreateKeyPair_InvalidPEM(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "invalid.key")

	// Write invalid PEM data
	if err := os.WriteFile(keyPath, []byte("not valid pem data"), 0600); err != nil {
		t.Fatalf("failed to write invalid key: %v", err)
	}

	cfg := &config.Config{
		Auth: config.Auth{
			KeyPath: keyPath,
		},
	}

	_, err := loadOrCreateKeyPair(cfg)
	if err == nil {
		t.Error("expected error for invalid PEM data")
	}
}

func TestGenerateKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "generated.key")

	key, err := generateKeyPair(keyPath)
	if err != nil {
		t.Fatalf("generateKeyPair failed: %v", err)
	}

	if key == nil {
		t.Fatal("expected key to be non-nil")
	}

	// Verify key size (should be 2048 bits)
	if key.N.BitLen() != 2048 {
		t.Errorf("expected 2048-bit key, got %d bits", key.N.BitLen())
	}

	// Verify the file was created and is readable
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read key file: %v", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("failed to decode PEM block")
	}

	parsedKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse key: %v", err)
	}

	if parsedKey.N.Cmp(key.N) != 0 {
		t.Error("parsed key does not match generated key")
	}
}

func TestGenerateKeyPair_InvalidPath(t *testing.T) {
	// Try to create a key in a non-existent directory
	keyPath := "/nonexistent/directory/test.key"

	_, err := generateKeyPair(keyPath)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestKeyPairPublicMatchesPrivate(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "keypair.key")

	cfg := &config.Config{
		Auth: config.Auth{
			KeyPath: keyPath,
		},
	}

	kp, err := loadOrCreateKeyPair(cfg)
	if err != nil {
		t.Fatalf("loadOrCreateKeyPair failed: %v", err)
	}

	// Verify public key matches private key's public component
	if kp.public.N.Cmp(kp.private.N) != 0 {
		t.Error("public key N does not match private key N")
	}

	if kp.public.E != kp.private.E {
		t.Error("public key E does not match private key E")
	}
}
