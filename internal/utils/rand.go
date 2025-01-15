package utils

import "crypto/rand"

const randDict = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!@#$%^&*_+-=~"

// RandStr returns random string of some size
func RandStr(strSize int) string {
	var bytes = make([]byte, strSize)
	_, _ = rand.Read(bytes)
	for k, v := range bytes {
		bytes[k] = randDict[v%byte(len(randDict))]
	}
	return string(bytes)
}
