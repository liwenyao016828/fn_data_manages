package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"regexp"
)

const encryptionPassphrase = "dm-key-2026-secure"

var encryptionKey []byte

func initCrypto() {
	passphrase := os.Getenv("DM_ENCRYPT_KEY")
	if passphrase == "" {
		passphrase = "dm-key-2026-secure"
	}
	h := sha256.Sum256([]byte(passphrase))
	encryptionKey = h[:]
}

func encryptPassword(plaintext string) string {
	if plaintext == "" {
		return ""
	}
	block, _ := aes.NewCipher(encryptionKey)
	aesGCM, _ := cipher.NewGCM(block)
	nonce := make([]byte, aesGCM.NonceSize())
	rand.Read(nonce)
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func decryptPassword(encoded string) string {
	if encoded == "" {
		return ""
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return encoded
	}
	block, _ := aes.NewCipher(encryptionKey)
	aesGCM, _ := cipher.NewGCM(block)
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return encoded
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return encoded
	}
	return string(plaintext)
}

func isPasswordEncrypted(s string) bool {
	if s == "" {
		return false
	}
	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		return false
	}
	return len(s) > 32
}

func validateIdentity(s string) string {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-\.]+$`, s)
	if !matched {
		return ""
	}
	return s
}