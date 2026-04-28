package client

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

// Crypto layout strictly follows internal/crypto/crypto.go on the panel side:
//   ciphertext = base64( salt(16) || nonce(12) || aesgcm.Seal(plaintext_with_8byte_LE_unix_ts) )
//   key        = PBKDF2-SHA256(passphrase=string(hex.Decode(commKeyHex)), salt=salt, iter=100000, keylen=32)

const (
	saltSize   = 16
	nonceSize  = 12
	keySize    = 32
	iterations = 100000
	maxAge     = 300
)

func deriveKey(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, iterations, keySize, sha256.New)
}

// Encrypt mirrors panel's crypto.Encrypt.
func Encrypt(plaintext []byte, hexKey string) (string, error) {
	rawKey, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", fmt.Errorf("invalid hex key: %w", err)
	}

	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	derived := deriveKey(string(rawKey), salt)

	block, err := aes.NewCipher(derived)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ts := time.Now().Unix()
	tsBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		tsBytes[i] = byte(ts >> (i * 8))
	}

	payload := append(tsBytes, plaintext...)
	ciphertext := aesGCM.Seal(nil, nonce, payload, nil)

	out := make([]byte, 0, saltSize+nonceSize+len(ciphertext))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt mirrors panel's crypto.Decrypt.
func Decrypt(encoded string, hexKey string) ([]byte, error) {
	rawKey, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid base64: %w", err)
	}
	if len(data) < saltSize+nonceSize+8 {
		return nil, errors.New("ciphertext too short")
	}
	salt := data[:saltSize]
	nonce := data[saltSize : saltSize+nonceSize]
	ciphertext := data[saltSize+nonceSize:]

	derived := deriveKey(string(rawKey), salt)
	block, err := aes.NewCipher(derived)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	payload, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	if len(payload) < 8 {
		return nil, errors.New("invalid payload")
	}
	var ts int64
	for i := 0; i < 8; i++ {
		ts |= int64(payload[i]) << (i * 8)
	}
	age := time.Now().Unix() - ts
	if age < -maxAge || age > maxAge {
		return nil, fmt.Errorf("message expired (age: %ds)", age)
	}
	return payload[8:], nil
}
