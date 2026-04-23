package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// DeriveKey pads/truncates a string key to exactly 32 bytes for AES-256.
func DeriveKey(key string) []byte {
	b := []byte(key)
	if len(b) >= 32 {
		return b[:32]
	}
	padded := make([]byte, 32)
	copy(padded, b)
	return padded
}

// GetEncryptionKey returns the 32-byte AES-256 key from environment.
// Falls back to JWT_SECRET if NOVA_ENCRYPTION_KEY is not set.
// The key is derived by padding/truncating to exactly 32 bytes.
func GetEncryptionKey() ([]byte, error) {
	key := os.Getenv("NOVA_ENCRYPTION_KEY")
	if key == "" {
		key = os.Getenv("JWT_SECRET")
	}
	if key == "" {
		return nil, fmt.Errorf("no encryption key available: set NOVA_ENCRYPTION_KEY or JWT_SECRET")
	}

	// Ensure exactly 32 bytes for AES-256
	keyBytes := []byte(key)
	if len(keyBytes) >= 32 {
		return keyBytes[:32], nil
	}

	// Pad with zeros if too short
	padded := make([]byte, 32)
	copy(padded, keyBytes)
	return padded, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded string.
// The nonce is prepended to the ciphertext.
func Encrypt(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded AES-256-GCM ciphertext string.
func Decrypt(encoded string, key []byte) (string, error) {
	if encoded == "" {
		return "", nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// If decoding fails, the value might be plaintext (pre-migration)
		return encoded, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		// Too short to be encrypted — return as-is (plaintext fallback)
		return encoded, nil
	}

	nonce, ciphertextBytes := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		// Decryption failed — might be plaintext pre-migration data
		return encoded, nil
	}

	return string(plaintext), nil
}
