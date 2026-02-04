package domain

import (
	"time"

	"github.com/samber/lo"
)

type Namespace struct {
	Name      string         `json:"name"`
	Network   string         `json:"network"`
	NetworkId string         `json:"network_id"`
	Subnet    string         `json:"subnet"`
	Gateway   string         `json:"gateway"`
	CreatedAt time.Time      `json:"created_at"`
	Redis     []RedisSpec    `json:"redis"`
	Database  []DatabaseSpec `json:"database"`
	App       []AppSpec      `json:"app"`
}

func (n *Namespace) FindApp(appName string) (AppSpec, bool) {
	return lo.Find(n.App, func(app AppSpec) bool {
		return appName == app.Name
	})
}
