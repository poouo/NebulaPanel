package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

const (
	NonceSize  = 12
	KeySize    = 32
	SaltSize   = 16
	Iterations = 100000
	MaxAge     = 300 // message max age in seconds (5 min)
)

// DeriveKey derives a 256-bit key from a passphrase using PBKDF2
func DeriveKey(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, Iterations, KeySize, sha256.New)
}

// GenerateKey generates a random communication key (hex encoded)
func GenerateKey() (string, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

// Encrypt encrypts plaintext with the given hex key using AES-256-GCM
// Format: base64(salt + nonce + timestamp(8bytes) + ciphertext + tag)
func Encrypt(plaintext []byte, hexKey string) (string, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", fmt.Errorf("invalid hex key: %w", err)
	}

	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	derivedKey := DeriveKey(string(key), salt)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Add timestamp to prevent replay attacks
	ts := time.Now().Unix()
	tsBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		tsBytes[i] = byte(ts >> (i * 8))
	}

	// Prepend timestamp to plaintext before encryption
	payload := append(tsBytes, plaintext...)
	ciphertext := aesGCM.Seal(nil, nonce, payload, nil)

	// Final format: salt + nonce + ciphertext
	result := make([]byte, 0, SaltSize+NonceSize+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return base64.StdEncoding.EncodeToString(result), nil
}

// Decrypt decrypts ciphertext with the given hex key
func Decrypt(encrypted string, hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("invalid base64: %w", err)
	}

	if len(data) < SaltSize+NonceSize+8 {
		return nil, errors.New("ciphertext too short")
	}

	salt := data[:SaltSize]
	nonce := data[SaltSize : SaltSize+NonceSize]
	ciphertext := data[SaltSize+NonceSize:]

	derivedKey := DeriveKey(string(key), salt)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	payload, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong key?): %w", err)
	}

	if len(payload) < 8 {
		return nil, errors.New("invalid payload")
	}

	// Extract and verify timestamp
	var ts int64
	for i := 0; i < 8; i++ {
		ts |= int64(payload[i]) << (i * 8)
	}
	age := time.Now().Unix() - ts
	if age < -MaxAge || age > MaxAge {
		return nil, fmt.Errorf("message expired (age: %ds)", age)
	}

	return payload[8:], nil
}
