// http://www.jasypt.org/ alternative for go
//
// Deprecated: legacy Jasypt PBEWithMD5AndDES compatibility; do not use for new secrets.
package jasypt

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

func getDerivedKey(password string, salt string, count int) ([]byte, []byte) {
	key := md5.Sum([]byte(password + salt))
	for i := 0; i < count-1; i++ {
		key = md5.Sum(key[:])
	}
	return key[:8], key[8:]
}

func doEncrypt(password, plainText string, salt []byte, iterationCount int) ([]byte, error) {
	padNum := byte(8 - len(plainText)%8)
	padded := make([]byte, 0, len(plainText)+int(padNum))
	padded = append(padded, plainText...)
	padded = append(padded, bytes.Repeat([]byte{padNum}, int(padNum))...)

	dk, iv := getDerivedKey(password, string(salt), iterationCount)
	block, err := des.NewCipher(dk)
	if err != nil {
		return nil, err
	}

	encrypter := cipher.NewCBCEncrypter(block, iv)
	encrypted := make([]byte, len(padded))
	encrypter.CryptBlocks(encrypted, padded)

	return encrypted, nil
}

func doDecrypt(password string, encText, salt []byte, iterationCount int) (string, error) {
	if len(encText) == 0 || len(encText)%8 != 0 {
		return "", errors.New("invalid ciphertext length")
	}

	dk, iv := getDerivedKey(password, string(salt), iterationCount)
	block, err := des.NewCipher(dk)

	if err != nil {
		return "", err
	}

	decrypter := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encText))
	decrypter.CryptBlocks(decrypted, encText)

	padNum := int(decrypted[len(decrypted)-1])
	if padNum < 1 || padNum > 8 {
		return "", errors.New("invalid padding")
	}

	return string(decrypted[:len(decrypted)-padNum]), nil
}

func Encrypt(password string, iterationCount int, plainText string) (string, error) {
	salt := make([]byte, 8)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	encText, err := doEncrypt(password, plainText, salt, iterationCount)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(append(salt, encText...)), nil
}

func Decrypt(password string, iterationCount int, cipherText string) (string, error) {
	msgBytes, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	if len(msgBytes) < 8 || len(msgBytes)%8 != 0 {
		return "", errors.New("invalid ciphertext length")
	}

	salt := msgBytes[:8]
	encText := msgBytes[8:]
	return doDecrypt(password, encText, salt, iterationCount)
}
