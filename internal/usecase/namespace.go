package usecase

import (
	"errors"

	"dockflow/internal/domain"
	"dockflow/internal/service/docker"
	"dockflow/internal/util"
)

var (
	ErrNamespaceNotFound = errors.New("namespace not found")
	ErrNamespaceExists   = errors.New("namespace already exists")
)

func CreateNamespace(name string) (*domain.Namespace, error) {

	namespace, err := domain.NewNamespace(name)
	if err == nil && namespace != nil {
		return nil, ErrNamespaceExists
	}

	namespaces := domain.ListNamespaces()

	subnet, gateway, err := util.AllocateSubnet(namespaces)
	if err != nil {
		return nil, err
	}
	ns := &domain.Namespace{
		Name:    name,
		Network: "dockflow_" + name,
		Subnet:  subnet,
		Gateway: gateway,
	}

	namespaceNetworkOpts := docker.NetworkCreateOptions{
		Name:       ns.Network,
		Driver:     "bridge",
		Subnet:     ns.Subnet,
		Gateway:    ns.Gateway,
		Attachable: true,
	}
	networkId, err := docker.CreateNetwork(namespaceNetworkOpts)
	if err != nil {
		return nil, err
	}
	ns.NetworkId = networkId

	if err := ns.Save(); err != nil {
		return nil, err
	}

	return ns, nil
}

func ListNamespace() []domain.Namespace {
	return domain.ListNamespaces()
}

func InspectNamespace(
	name string,
) (*domain.Namespace, error) {

	ns, err := domain.NewNamespace(name)
	if err != nil {
		// if filesystem.DirExists() {
		// 	return nil, ErrNamespaceNotFound
		// }
		return nil, err
	}
	return ns, nil
}

func RemoveNamespace(name string) error {

	ns, err := domain.NewNamespace(name)
	if err != nil {
		return ErrNamespaceNotFound
	}

	// 先删 docker network
	if err := docker.RemoveNetwork(ns.NetworkId); err != nil {
		return err
	}

	// 再删本地状态
	return ns.Remove()
}
