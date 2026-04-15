package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

// NewAESGCMFromKeyString creates AES-GCM from a key string.
// Preferred input: base64(32 bytes) or raw 32 bytes.
// Backward-compatible fallback: derive a 32-byte key from any non-empty string via SHA-256.
func NewAESGCMFromKeyString(key string) (cipher.AEAD, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("encryption_key is required")
	}

	kb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		// fallback: treat as raw bytes
		kb = []byte(key)
	}
	if len(kb) != 32 {
		sum := sha256.Sum256([]byte(key))
		kb = sum[:]
	}

	block, err := aes.NewCipher(kb)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func EncryptString(aead cipher.AEAD, plaintext string) (string, error) {
	if aead == nil {
		return "", errors.New("aead is nil")
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := aead.Seal(nil, nonce, []byte(plaintext), nil)
	out := append(nonce, ct...)
	return base64.StdEncoding.EncodeToString(out), nil
}

func DecryptString(aead cipher.AEAD, ciphertextB64 string) (string, error) {
	if aead == nil {
		return "", errors.New("aead is nil")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(ciphertextB64))
	if err != nil {
		return "", err
	}
	ns := aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
