package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type NetworkManager struct {
	cli *client.Client
}

func NewNetworkManager(cli *client.Client) *NetworkManager {
	return &NetworkManager{cli: cli}
}

/* ---------- 创建 ---------- */

type CreateNetworkOptions struct {
	Name     string
	Driver   string
	Internal bool
	Labels   map[string]string
}

func (m *NetworkManager) Create(
	ctx context.Context,
	opt CreateNetworkOptions,
) (string, error) {

	if opt.Driver == "" {
		opt.Driver = "bridge"
	}

	resp, err := m.cli.NetworkCreate(ctx, opt.Name, types.NetworkCreate{
		Driver:   opt.Driver,
		Internal: opt.Internal,
		Labels:   opt.Labels,
	})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

/* ---------- 管理 ---------- */

func (m *NetworkManager) Connect(
	ctx context.Context,
	networkID string,
	containerID string,
) error {
	return m.cli.NetworkConnect(ctx, networkID, containerID, &network.EndpointSettings{})
}

func (m *NetworkManager) Disconnect(
	ctx context.Context,
	networkID string,
	containerID string,
) error {
	return m.cli.NetworkDisconnect(ctx, networkID, containerID, false)
}

func (m *NetworkManager) List(
	ctx context.Context,
	f filters.Args,
) ([]types.NetworkResource, error) {
	return m.cli.NetworkList(ctx, types.NetworkListOptions{Filters: f})
}

func (m *NetworkManager) Prune(
	ctx context.Context,
) (types.NetworksPruneReport, error) {

	f := filters.NewArgs()
	f.Add("label", "dockflow.managed=true")

	return m.cli.NetworksPrune(ctx, f)
}
