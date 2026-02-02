package util

import (
	"math/rand"
	"time"
)

func GenerateRandomPort() int {
	// 设置随机种子
	rand.Seed(time.Now().UnixNano())

	// 生成10000到65535之间的随机数
	min := 10000
	max := 65535

	return rand.Intn(max-min+1) + min
}
