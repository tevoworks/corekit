package webhook

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

func encryptAESGCM(key []byte, plaintext string) (string, error) {
	if len(key) == 0 {
		return plaintext, nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesgcm.Seal(nil, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(nonce) + ":" + hex.EncodeToString(ciphertext), nil
}

func decryptAESGCM(key []byte, encoded string) (string, error) {
	if len(key) == 0 {
		return encoded, nil
	}
	parts := splitEncrypted(encoded)
	if len(parts) != 2 {
		return "", errors.New("invalid encrypted format")
	}
	nonce, err := hex.DecodeString(parts[0])
	if err != nil {
		return "", err
	}
	ciphertext, err := hex.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func splitEncrypted(s string) []string {
	var parts []string
	var buf []byte
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			parts = append(parts, string(buf))
			buf = nil
			continue
		}
		buf = append(buf, s[i])
	}
	if len(buf) > 0 {
		parts = append(parts, string(buf))
	}
	return parts
}
