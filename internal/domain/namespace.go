package domain

import "time"

type Namespace struct {
	Name      string
	Network   string
	NetworkId string
	Subnet    string
	Gateway   string
	CreatedAt time.Time
}
