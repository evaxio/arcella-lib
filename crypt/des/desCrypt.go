// Deprecated: legacy single-DES interop only; do not use for new data.
package aes

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"errors"
)

func DesDecrypt(crypted, key []byte) ([]byte, error) {
	if len(crypted) == 0 || len(crypted)%8 != 0 {
		return nil, errors.New("ciphertext must be a non-empty multiple of 8 bytes")
	}
	if block, err := des.NewCipher(key); err == nil {
		bs := block.BlockSize()
		startData := make([]byte, bs)
		origData := make([]byte, len(crypted))

		block.Decrypt(startData, crypted[:bs]) // 0..7

		blockMode := cipher.NewCBCDecrypter(block, key) // other
		blockMode.CryptBlocks(origData, crypted)
		origData = ZeroUnPadding(origData)
		for i := 0; i < bs; i++ {
			origData[i] = startData[i]
		}
		return origData, nil
	} else {
		return nil, err
	}
}

// ZeroUnPadding trims all trailing zero bytes; the plaintext must not end
// with NUL bytes, or the trailing data will be lost.
func ZeroUnPadding(origData []byte) []byte {
	return bytes.TrimFunc(origData, func(r rune) bool {
		return r == rune(0)
	})
}
