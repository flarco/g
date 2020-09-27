package gutil

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	SaltByteSize     = 24
	HashByteSize     = 24
	PBKDF2Iterations = 1000
)

// Hash hashes a password
func Hash(password string) (string, error) {
	salt := make([]byte, SaltByteSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		fmt.Print("Err generating random salt")
		return "", errors.New("Err generating random salt")
	}

	//todo: enhance: randomize itrs as well
	hbts := pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, HashByteSize, sha1.New)
	//hbtstr := fmt.Sprintf("%x", hbts)

	return fmt.Sprintf("%v:%v:%v",
		PBKDF2Iterations,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(hbts)), nil
}

// VerifyHash verifies the hash
func VerifyHash(raw, hash string) (bool, error) {
	hparts := strings.Split(hash, ":")

	itr, err := strconv.Atoi(hparts[0])
	if err != nil {
		fmt.Printf("wrong hash %v", hash)
		return false, errors.New("wrong hash, iteration is invalid")
	}
	salt, err := base64.StdEncoding.DecodeString(hparts[1])
	if err != nil {
		fmt.Print("wrong hash, salt error:", err)
		return false, errors.New("wrong hash, salt error:" + err.Error())
	}

	hsh, err := base64.StdEncoding.DecodeString(hparts[2])
	if err != nil {
		fmt.Print("wrong hash, hash error:", err)
		return false, errors.New("wrong hash, hash error:" + err.Error())
	}

	rhash := pbkdf2.Key([]byte(raw), salt, itr, len(hsh), sha1.New)
	return hashEqual(rhash, hsh), nil
}

//bytes comparisons
func hashEqual(h1, h2 []byte) bool {
	diff := uint32(len(h1)) ^ uint32(len(h2))
	for i := 0; i < len(h1) && i < len(h2); i++ {
		diff |= uint32(h1[i] ^ h2[i])
	}

	return diff == 0
}
