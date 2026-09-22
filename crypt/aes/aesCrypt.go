package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

func Crypt(text, secret string) (string, error) {
	cBlock, err := aes.NewCipher(createHash(secret))
	if err != nil {
		return "", err
	}
	return CryptPass(text, &cBlock, true)
}

func DeCrypt(text, secret string) (string, error) {
	cBlock, err := aes.NewCipher(createHash(secret))
	if err != nil {
		return "", err
	}
	return CryptPass(text, &cBlock, false)
}

func CryptPass(text string, cBlock *cipher.Block, isEncrypt bool) (string, error) {
	var err error
	var gcm cipher.AEAD
	if gcm, err = cipher.NewGCM(*cBlock); err == nil {
		if isEncrypt {
			nonce := make([]byte, gcm.NonceSize())
			if _, err = io.ReadFull(rand.Reader, nonce); err == nil {
				return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(text), nil)), nil
			} else {
				return "", err
			}
		} else { // Decrypt
			nonceSize := gcm.NonceSize()
			var data []byte
			if data, err = base64.StdEncoding.DecodeString(text); err == nil {
				if len(data) < nonceSize {
					return "", errors.New("ciphertext too short")
				}
				nonce, ciphertext := data[:nonceSize], data[nonceSize:]
				var byteText []byte
				if byteText, err = gcm.Open(nil, nonce, ciphertext, nil); err == nil {
					return string(byteText), nil
				} else {
					return "", err
				}
			} else {
				return "", err
			}
		}
	} else {
		return "", err
	}
}

func createHash(key string) []byte {
	sha256s := sha256.Sum256([]byte(key))
	return sha256s[:]
}
