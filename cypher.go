package gutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

// EncryptInBytes some data, not friendly over the wire
func EncryptInBytes(data []byte, passphrase string) []byte {
	block, _ := aes.NewCipher([]byte(createHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(crand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext
}

// DecryptInByte some data, not friendly over the wire
func DecryptInByte(data []byte, passphrase string) []byte {
	key := []byte(createHash(passphrase))
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}
	return plaintext
}

// Encrypt string to base64 crypto using AES
// the key should be 16, 24 or 32 byte
func Encrypt(text string, key string) (string, error) {
	if text == "" {
		return "", nil
	}
	// key := []byte(keyText)
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		err := fmt.Errorf("key needs to be of length 16, 24 or 32 chars")
		return "", Error(err)
	}
	plaintext := []byte(text)

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", Error(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(crand.Reader, iv); err != nil {
		return "", Error(err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// convert to base64
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt from base64 to decrypted string
func Decrypt(cryptoText string, key string) (string, error) {
	if cryptoText == "" {
		return "", nil
	}

	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		err := fmt.Errorf("key needs to be of length 16, 24 or 32 chars")
		return "", Error(err)
	}
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", Error(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		err := fmt.Errorf("ciphertext too short")
		return "", Error(err)
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext), nil
}
