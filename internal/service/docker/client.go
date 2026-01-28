package docker

import (
	"context"
	"sync"

	"github.com/docker/docker/client"
)

var (
	cli  *client.Client
	once sync.Once
)

// Client 获取全局唯一 Docker Client
func Client() *client.Client {
	once.Do(func() {
		c, err := client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			panic(err) // CLI 工具：启动即失败是合理的
		}
		cli = c
	})
	return cli
}

// Ctx 统一上下文（后续可扩展 timeout / trace）
func Ctx() context.Context {
	return context.Background()
}
