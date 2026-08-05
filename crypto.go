package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// A locked note stores its content as "GOPADENC1:" + base64(salt|nonce|ciphertext).
// The key is scrypt-derived from the user's passphrase; the GCM auth tag is what
// verifies the passphrase, so no password hash is stored anywhere.
const encPrefix = "GOPADENC1:"

const (
	saltLen = 16
	keyLen  = 32 // AES-256
	scryptN = 1 << 15
	scryptR = 8
	scryptP = 1
)

var (
	errBadPassphrase = errors.New("wrong passphrase")
	errNotEncrypted  = errors.New("note is not encrypted")
	errEmptyPass     = errors.New("passphrase cannot be empty")
)

func newSalt() ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

func deriveKey(passphrase string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, keyLen)
}

// sealContent encrypts plain under key. The salt is carried in the output so the
// key can be re-derived later; callers reuse a live key rather than paying the
// ~100ms scrypt cost on every autosave.
func sealContent(plain string, key, salt []byte) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nil, nonce, []byte(plain), nil)

	blob := make([]byte, 0, len(salt)+len(nonce)+len(ct))
	blob = append(blob, salt...)
	blob = append(blob, nonce...)
	blob = append(blob, ct...)
	return encPrefix + base64.StdEncoding.EncodeToString(blob), nil
}

// openContent decrypts stored with key. A failed auth tag means the wrong
// passphrase was used.
func openContent(stored string, key []byte) (string, error) {
	_, nonce, ct, err := splitEncrypted(stored)
	if err != nil {
		return "", err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	if len(nonce) != gcm.NonceSize() {
		return "", errNotEncrypted
	}
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", errBadPassphrase
	}
	return string(plain), nil
}

// saltOf extracts the salt a locked note was encrypted with, so a passphrase can
// be turned back into the right key.
func saltOf(stored string) ([]byte, error) {
	salt, _, _, err := splitEncrypted(stored)
	return salt, err
}

func isEncrypted(stored string) bool {
	return strings.HasPrefix(stored, encPrefix)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func splitEncrypted(stored string) (salt, nonce, ct []byte, err error) {
	if !isEncrypted(stored) {
		return nil, nil, nil, errNotEncrypted
	}
	blob, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, encPrefix))
	if err != nil {
		return nil, nil, nil, errNotEncrypted
	}
	// salt + 12-byte GCM nonce + at least the 16-byte auth tag
	if len(blob) < saltLen+12+16 {
		return nil, nil, nil, errNotEncrypted
	}
	return blob[:saltLen], blob[saltLen : saltLen+12], blob[saltLen+12:], nil
}
