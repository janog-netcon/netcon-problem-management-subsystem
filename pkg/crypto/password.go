package crypto

import (
	"crypto/rand"
	"errors"
	"math/big"
)

func GeneratePassword(length int) (string, error) {
	if length < 0 {
		return "", errors.New("invalid length")
	}

	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	var buf = make([]rune, length)
	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		buf[i] = chars[idx.Int64()]
	}

	return string(buf), nil
}
