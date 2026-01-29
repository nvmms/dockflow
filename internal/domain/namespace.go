package domain

import "time"

type Namespace struct {
	Name      string         `json:"name"`
	Network   string         `json:"network"`
	NetworkId string         `json:"network_id"`
	Subnet    string         `json:"subnet"`
	Gateway   string         `json:"gateway"`
	CreatedAt time.Time      `json:"created_at"`
	Redis     []RedisSpec    `json:"redis"`
	Database  []DatabaseSpec `json:"database"`
}
