package usecase

import (
	"fmt"
	"io"

	"dockflow/internal/domain"
	"dockflow/internal/service/docker"

	"github.com/samber/lo"
)

// OpenAppRuntimeLogs validates ownership before exposing a Docker log stream.
func OpenAppRuntimeLogs(namespace, appName, containerID, tail string) (io.ReadCloser, error) {
	ns, err := domain.NewNamespace(namespace)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		return nil, ErrNamespaceNotFound
	}
	app, found := ns.FindApp(appName)
	if !found {
		return nil, ErrAppNotFound
	}
	if _, found = lo.Find(app.Deploy, func(item domain.AppDeploy) bool { return item.ContainerId == containerID }); !found {
		return nil, fmt.Errorf("deployment not found")
	}
	return openContainerLogs(containerID, tail)
}

func OpenDatabaseLogs(namespace, name, tail string) (io.ReadCloser, error) {
	database, err := findDatabase(namespace, name)
	if err != nil {
		return nil, err
	}
	return openContainerLogs(database.ContainerId, tail)
}

func OpenRedisLogs(namespace, name, tail string) (io.ReadCloser, error) {
	ns, err := domain.NewNamespace(namespace)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		return nil, ErrNamespaceNotFound
	}
	redis, _ := findRedisByName(ns, name)
	if redis == nil {
		return nil, fmt.Errorf("redis [%s] not exist", name)
	}
	return openContainerLogs(redis.ContainerId, tail)
}

func openContainerLogs(containerID, tail string) (io.ReadCloser, error) {
	if tail == "" {
		tail = "200"
	}
	return docker.ContainerLogs(containerID, docker.ContainerLogOptions{Follow: true, Tail: tail})
}
