package usecase

import (
	"dockflow/internal/domain"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/filesystem"
	"errors"
)

var (
	ErrRedisNotFound = errors.New("redis not found")
	ErrRedisNotExist = errors.New("redis not exist")
	ErrRedisExist    = errors.New("redis name is exist")
)

func CreateRedis(redis domain.RedisSpec) error {

	ns, err := filesystem.LoadNamespace(redis.Namespace)
	if err != nil {
		return err
	}
	if ns == nil {
		return ErrNamespaceNotFound
	}

	currentRedis, currentRedisIndex := findRedisByName(ns, redis.Name)
	if currentRedis != nil || currentRedisIndex > -1 {
		return ErrRedisExist
	}

	redisImageName := "redis:" + redis.Version
	if err := docker.PullImage(redisImageName); err != nil {
		return err
	}

	opts := docker.NewRunOptions(redis.Name, redisImageName)

	opts.WithNetwork(ns.Network)
	opts.WithCpu(redis.CPU)
	opts.WithMemory(redis.Memory)

	var aof = "yes"
	if !redis.AOF {
		aof = "no"
	}
	opts.WithCommand(
		"redis-server",
		"--requirepass",
		redis.Password,
		"--appendonly",
		aof,
		"--maxmemory-policy",
		redis.Eviction,
	)

	containerId, err := docker.RunContainer(opts)
	if err != nil {
		return err
	}

	inspect, err := docker.InspectContainer(containerId)
	if err != nil {
		return err
	}

	ips := []string{}
	for _, net := range inspect.NetworkSettings.Networks {
		ips = append(ips, net.IPAddress)
	}

	redis.ContainerId = containerId
	redis.Ip = ips
	ns.Redis = append(ns.Redis, redis)
	filesystem.SaveNamespace(*ns)

	return nil
}

func ListRedis(namespaceName string) ([]domain.RedisSpec, error) {
	ns, err := filesystem.LoadNamespace(namespaceName)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		return nil, ErrNamespaceNotFound
	}

	return ns.Redis, nil
}

func RemoveRedis(namespaceName string, redisContainerName string) error {
	ns, err := filesystem.LoadNamespace(namespaceName)
	if err != nil {
		return err
	}
	if ns == nil {
		return ErrNamespaceNotFound
	}

	redis, index := findRedisByName(ns, redisContainerName)
	if redis == nil || index <= -1 {
		return ErrRedisNotExist
	}

	containerId, err := docker.HasContainer(redis.ContainerId)
	if err != nil {
		return err
	}
	if containerId == "" {
		return ErrRedisNotExist
	}

	isRun, err := docker.ContainerRunning(redis.ContainerId)
	if err != nil {
		return err
	}
	if isRun {
		err := docker.StopContainer(redis.ContainerId, nil)
		if err != nil {
			return err
		}
	}

	err = docker.RemoveContainer(redis.ContainerId, true)
	if err != nil {
		return err
	}

	ns.Redis = remove(ns.Redis, index)
	filesystem.SaveNamespace(*ns)

	return nil

}

func findRedisByName(ns *domain.Namespace, redisContainerName string) (*domain.RedisSpec, int) {
	for i, redis := range ns.Redis {
		if redis.Name == redisContainerName {
			return &redis, i
		}
	}
	return nil, -1
}

func remove(slice []domain.RedisSpec, i int) []domain.RedisSpec {
	return append(slice[:i], slice[i+1:]...)
}
