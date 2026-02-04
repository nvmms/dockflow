package util

import (
	"math/rand"
	"time"
)

const (
	letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits  = "0123456789"
	special = "!@#$%^&*()-_=+[]{}|;:,.<>?/~"
)

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	// 设置随机种子
	rand.NewSource(time.Now().UnixNano())

	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// GenerateRandomStringWithCharset 使用指定字符集生成随机字符串
func GenerateRandomStringWithCharset(length int, charset string) string {
	rand.NewSource(time.Now().UnixNano())

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
