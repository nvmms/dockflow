package usecase

import (
	"dockflow/internal/domain"
	"dockflow/internal/service/docker"
	"errors"
	"fmt"
)

var (
	ErrRedisNotFound = errors.New("redis not found")
	ErrRedisNotExist = errors.New("redis not exist")
	ErrRedisExist    = errors.New("redis name is exist")
)

func CreateRedis(redis domain.RedisSpec) error {
	ns, err := domain.NewNamespace(redis.Namespace)
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

	containerName := docker.ResourceContainerName(redis.Namespace, "redis", redis.Name)
	opts := docker.NewRunOptions(containerName, redisImageName)
	restartPolicy, err := normalizeRestartPolicy(redis.RestartPolicy)
	if err != nil {
		return err
	}
	redis.RestartPolicy = string(restartPolicy)
	opts.WithRestart(restartPolicy)

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
	ns.Save()

	return nil
}

func ListRedis(namespaceName string) ([]domain.RedisSpec, error) {
	ns, err := domain.NewNamespace(namespaceName)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		return nil, ErrNamespaceNotFound
	}

	result := append([]domain.RedisSpec(nil), ns.Redis...)
	for i := range result {
		status, ips, restartPolicy, err := containerRuntimeStatus(result[i].ContainerId)
		if err != nil {
			return nil, err
		}
		result[i].Status = status
		result[i].Ip = ips
		if restartPolicy != "" {
			result[i].RestartPolicy = restartPolicy
		}
	}
	return result, nil
}

func SetRedisRunning(namespaceName, name string, running bool) error {
	ns, err := domain.NewNamespace(namespaceName)
	if err != nil {
		return err
	}
	if ns == nil {
		return ErrNamespaceNotFound
	}
	redis, _ := findRedisByName(ns, name)
	if redis == nil {
		return ErrRedisNotExist
	}
	return setContainerRunning(redis.ContainerId, running)
}

func UpdateRedisRestartPolicy(namespaceName, name, value string) error {
	policy, err := normalizeRestartPolicy(value)
	if err != nil {
		return err
	}
	ns, err := domain.NewNamespace(namespaceName)
	if err != nil {
		return err
	}
	if ns == nil {
		return ErrNamespaceNotFound
	}
	redis, index := findRedisByName(ns, name)
	if redis == nil {
		return ErrRedisNotExist
	}
	containerID, err := docker.HasContainer(redis.ContainerId)
	if err != nil {
		return err
	}
	if containerID == "" {
		return fmt.Errorf("container not found")
	}
	if err := docker.UpdateContainerRestartPolicy(containerID, policy); err != nil {
		return err
	}
	ns.Redis[index].RestartPolicy = string(policy)
	return ns.Save()
}

func containerRuntimeStatus(containerID string) (string, []string, string, error) {
	existingID, err := docker.HasContainer(containerID)
	if err != nil {
		return "", nil, "", err
	}
	if existingID == "" {
		return "missing", nil, "", nil
	}
	inspect, err := docker.InspectContainer(existingID)
	if err != nil {
		return "", nil, "", err
	}
	status := "stopped"
	if inspect.State != nil {
		if inspect.State.Running {
			status = "running"
		} else if inspect.State.Status == "paused" || inspect.State.Status == "restarting" {
			status = inspect.State.Status
		}
	}
	ips := []string{}
	if status == "running" && inspect.NetworkSettings != nil {
		for _, network := range inspect.NetworkSettings.Networks {
			if network != nil && network.IPAddress != "" {
				ips = append(ips, network.IPAddress)
			}
		}
	}
	restartPolicy := ""
	if inspect.HostConfig != nil {
		restartPolicy = string(inspect.HostConfig.RestartPolicy.Name)
	}
	return status, ips, restartPolicy, nil
}

func setContainerRunning(containerID string, running bool) error {
	existingID, err := docker.HasContainer(containerID)
	if err != nil {
		return err
	}
	if existingID == "" {
		return fmt.Errorf("container not found")
	}
	if running {
		return docker.StartContainer(existingID)
	}
	return docker.StopContainer(existingID, nil)
}

func RemoveRedis(namespaceName string, redisContainerName string) error {
	ns, err := domain.NewNamespace(namespaceName)
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
	if containerId != "" {
		isRun, err := docker.ContainerRunning(redis.ContainerId)
		if err != nil {
			return err
		}
		if isRun {
			if err := docker.StopContainer(redis.ContainerId, nil); err != nil {
				return err
			}
		}
		if err := docker.RemoveContainer(redis.ContainerId, true); err != nil {
			return err
		}
	}

	ns.Redis = remove(ns.Redis, index)
	ns.Save()

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
