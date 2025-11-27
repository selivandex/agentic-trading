package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"

	"prometheus/pkg/errors"
)

// Encryptor handles AES-256-GCM encryption/decryption
type Encryptor struct {
	key []byte // 32 bytes for AES-256
}

// NewEncryptor creates a new encryptor with a base64-encoded 32-byte key
func NewEncryptor(base64Key string) (*Encryptor, error) {
	if base64Key == "" {
		return nil, errors.New("encryption key cannot be empty")
	}

	// Decode base64 key to raw bytes
	keyBytes, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode base64 encryption key")
	}

	if len(keyBytes) != 32 {
		return nil, errors.Newf("encryption key must be exactly 32 bytes for AES-256, got %d bytes", len(keyBytes))
	}

	return &Encryptor{key: keyBytes}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (e *Encryptor) Encrypt(plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	// Create GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Create random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt and prepend nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (e *Encryptor) Decrypt(ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// EncryptBytes encrypts byte slice
func (e *Encryptor) EncryptBytes(plaintext []byte) ([]byte, error) {
	return e.Encrypt(string(plaintext))
}

// DecryptBytes decrypts to byte slice
func (e *Encryptor) DecryptBytes(ciphertext []byte) ([]byte, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}
	return []byte(plaintext), nil
}
