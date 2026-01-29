package docker

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

/*
=====================
 Network 基础操作
=====================
*/

// ListNetworks 列出网络
func ListNetworks() ([]types.NetworkResource, error) {
	return Client().NetworkList(
		Ctx(),
		types.NetworkListOptions{},
	)
}

// InspectNetwork 获取网络详情
func InspectNetwork(id string) (types.NetworkResource, error) {
	return Client().NetworkInspect(
		Ctx(),
		id,
		types.NetworkInspectOptions{},
	)
}

// HasNetwork 判断网络是否存在（支持 name 或 id）
func HasNetwork(networkID string) (bool, error) {
	list, err := ListNetworks()
	if err != nil {
		return false, err
	}
	for _, item := range list {
		if item.ID == networkID || item.Name == networkID {
			return true, nil
		}
	}
	return false, nil
}

// RemoveNetwork 删除网络
func RemoveNetwork(id string) error {
	return Client().NetworkRemove(
		Ctx(),
		id,
	)
}

/*
=====================
 Network 创建
=====================
*/

// NetworkCreateOptions 对应 docker network create
type NetworkCreateOptions struct {
	Name       string
	Driver     string
	Internal   bool
	Attachable bool
	Labels     map[string]string
	Options    map[string]string
	Subnet     string
	Gateway    string
	IPRange    string
}

// CreateNetwork 创建网络
func CreateNetwork(opts NetworkCreateOptions) (string, error) {
	if opts.Driver == "" {
		opts.Driver = "bridge"
	}

	var ipam *network.IPAM
	if opts.Subnet != "" {
		cfg := network.IPAMConfig{
			Subnet:  opts.Subnet,
			Gateway: opts.Gateway,
			IPRange: opts.IPRange,
		}
		ipam = &network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{cfg},
		}
	}

	resp, err := Client().NetworkCreate(
		Ctx(),
		opts.Name,
		types.NetworkCreate{
			Driver:     opts.Driver,
			Internal:   opts.Internal,
			Attachable: opts.Attachable,
			Labels:     opts.Labels,
			Options:    opts.Options,
			IPAM:       ipam,
		},
	)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

/*
=====================
 容器 ↔ 网络
=====================
*/

// ConnectNetwork 将容器接入网络
func ConnectNetwork(networkID, containerID string) error {
	return Client().NetworkConnect(
		Ctx(),
		networkID,
		containerID,
		&network.EndpointSettings{},
	)
}

// DisconnectNetwork 将容器从网络移除
func DisconnectNetwork(networkID, containerID string, force bool) error {
	return Client().NetworkDisconnect(
		Ctx(),
		networkID,
		containerID,
		force,
	)
}
