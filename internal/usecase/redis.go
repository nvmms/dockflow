package usecase

import (
	"dockflow/internal/domain"
	"dockflow/internal/service/docker"
	"errors"
	"fmt"
	"strconv"
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
	logicalLogDriver := redis.LogDriver
	if logicalLogDriver == "" {
		logicalLogDriver = "local"
	}
	logConfig, err := normalizeLogConfig(logicalLogDriver, redis.LogMaxSize, redis.LogMaxFile)
	if err != nil {
		return err
	}
	redis.LogDriver, redis.LogMaxSize = logicalLogDriver, logConfig.Config["max-size"]
	redis.LogMaxFile, _ = strconv.Atoi(logConfig.Config["max-file"])
	opts.WithLogging(logConfig.Type, redis.LogMaxSize, redis.LogMaxFile)
	for key, value := range slsLabels(redis.LogDriver, redis.SLSProject, redis.SLSLogstore, redis.SLSEndpoint, redis.SLSConfigName) {
		opts.WithLabel(key, value)
	}
	opts.WithVolume(fmt.Sprintf("dockflow-redisvolume-%s-%s", redis.Namespace, redis.Name), "/data")

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
		status, ips, restartPolicy, logDriver, logSize, logFile, slsEnabled, err := containerRuntimeStatus(result[i].ContainerId)
		if err != nil {
			return nil, err
		}
		result[i].Status = status
		result[i].Ip = ips
		if restartPolicy != "" {
			result[i].RestartPolicy = restartPolicy
		}
		desiredDriver := result[i].LogDriver
		if desiredDriver == "aliyun-sls" {
			desiredDriver = "json-file"
		}
		result[i].NeedsRecreate = result[i].LogDriver != "" && (desiredDriver != logDriver || result[i].LogMaxSize != logSize || result[i].LogMaxFile != logFile || (result[i].LogDriver == "aliyun-sls") != slsEnabled)
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

func UpdateRedisConfig(namespaceName, name string, edit ContainerEditOptions) error {
	policy, err := normalizeRestartPolicy(edit.RestartPolicy)
	if err != nil {
		return err
	}
	logConfig, err := normalizeLogConfig(edit.LogDriver, edit.LogMaxSize, edit.LogMaxFile)
	if err != nil {
		return err
	}
	if edit.LogDriver == "aliyun-sls" && (edit.SLSProject == "" || edit.SLSLogstore == "" || edit.SLSEndpoint == "" || edit.SLSConfigName == "") {
		return fmt.Errorf("Aliyun SLS configuration is incomplete")
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
	if edit.ApplyNow {
		inspect, err := docker.InspectContainer(containerID)
		if err != nil {
			return err
		}
		hasDataVolume := false
		for _, mount := range inspect.Mounts {
			if mount.Destination == "/data" {
				hasDataVolume = true
				break
			}
		}
		if !hasDataVolume {
			return fmt.Errorf("legacy Redis container has no persistent /data volume; immediate recreation is unsafe")
		}
		containerID, err = docker.RecreateContainer(containerID, policy, logConfig, slsContainerLabels(edit))
		if err != nil {
			return err
		}
	} else if err := docker.UpdateContainerRestartPolicy(containerID, policy); err != nil {
		return err
	}
	ns.Redis[index].RestartPolicy = string(policy)
	ns.Redis[index].LogDriver = edit.LogDriver
	ns.Redis[index].SLSProject, ns.Redis[index].SLSLogstore, ns.Redis[index].SLSEndpoint, ns.Redis[index].SLSConfigName = edit.SLSProject, edit.SLSLogstore, edit.SLSEndpoint, edit.SLSConfigName
	ns.Redis[index].LogMaxSize = logConfig.Config["max-size"]
	ns.Redis[index].LogMaxFile, _ = strconv.Atoi(logConfig.Config["max-file"])
	if edit.ApplyNow {
		ns.Redis[index].ContainerId = containerID
	}
	return ns.Save()
}

func containerRuntimeStatus(containerID string) (string, []string, string, string, string, int, bool, error) {
	existingID, err := docker.HasContainer(containerID)
	if err != nil {
		return "", nil, "", "", "", 0, false, err
	}
	if existingID == "" {
		return "missing", nil, "", "", "", 0, false, nil
	}
	inspect, err := docker.InspectContainer(existingID)
	if err != nil {
		return "", nil, "", "", "", 0, false, err
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
	logDriver, logSize, logFile := "", "", 0
	if inspect.HostConfig != nil {
		logDriver = inspect.HostConfig.LogConfig.Type
		logSize = inspect.HostConfig.LogConfig.Config["max-size"]
		logFile, _ = strconv.Atoi(inspect.HostConfig.LogConfig.Config["max-file"])
	}
	slsEnabled := inspect.Config != nil && inspect.Config.Labels["dockflow.sls.enabled"] == "true"
	return status, ips, restartPolicy, logDriver, logSize, logFile, slsEnabled, nil
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
