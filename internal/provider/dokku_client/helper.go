package dokkuclient

import (
	"fmt"
	"math/rand"
)

func DoubleDashArg[T string](key string, value T) string {
	return fmt.Sprintf("--%s %s", key, value)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
