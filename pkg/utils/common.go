package utils

import (
	"math/rand"
	"strings"
)

func RandStringRunes(n int) string {
	var letterRunes = []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func Replace(old string, r map[string]string) (new string) {
	new = old
	for k, v := range r {
		new = strings.Replace(new, k, v, -1)
	}
	return
}
