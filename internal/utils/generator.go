package utils

import (
	"crypto/rand"
	"math/big"
)

const (
	DefaultShortCodeLength = 6
	alphabet               = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

func GenerateShortCode() (string, error) {
	return GenerateShortCodeWithLength(DefaultShortCodeLength)
}

func GenerateShortCodeWithLength(length int) (string, error) {
	code := make([]byte, length)
	alphabetLen := big.NewInt(int64(len(alphabet)))

	for i := range code {
		randomIndex, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		code[i] = alphabet[randomIndex.Int64()]
	}

	return string(code), nil
}
