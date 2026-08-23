package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/samber/lo"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type ContainerLogOptions struct {
	Follow bool
	Tail   string
}

// ListContainers 列出容器
func ListContainers(all bool) ([]types.Container, error) {
	return Client().ContainerList(
		Ctx(),
		container.ListOptions{All: all},
	)
}

func HasContainer(containerId string) (string, error) {
	list, err := ListContainers(true)
	if err != nil {
		return "", err
	}
	for _, item := range list {
		if item.ID == containerId || lo.Contains(item.Names, "/"+containerId) {
			return item.ID, nil
		}
	}
	return "", nil
}

func ContainerRunning(containerId string) (bool, error) {
	info, err := InspectContainer(containerId)
	if err != nil {
		return false, err
	}
	if info.State == nil {
		return false, nil
	}
	return info.State.Running, nil
}

// InspectContainer 获取容器详情
func InspectContainer(id string) (types.ContainerJSON, error) {
	return Client().ContainerInspect(Ctx(), id)
}

// StartContainer 启动容器
func StartContainer(id string) error {
	return Client().ContainerStart(
		Ctx(),
		id,
		container.StartOptions{},
	)
}

// StopContainer 停止容器
func StopContainer(id string, timeoutSec *int) error {
	return Client().ContainerStop(
		Ctx(),
		id,
		container.StopOptions{Timeout: timeoutSec},
	)
}

// RestartContainer 重启容器
func RestartContainer(id string, timeoutSec *int) error {
	return Client().ContainerRestart(
		Ctx(),
		id,
		container.StopOptions{Timeout: timeoutSec},
	)
}

// UpdateContainerRestartPolicy changes the policy of an existing container.
// Docker applies this immediately; restarting the container is not required.
func UpdateContainerRestartPolicy(id string, mode container.RestartPolicyMode) error {
	_, err := Client().ContainerUpdate(
		Ctx(),
		id,
		container.UpdateConfig{RestartPolicy: container.RestartPolicy{Name: mode}},
	)
	return err
}

// RemoveContainer 删除容器
func RemoveContainer(id string, force bool) error {
	return Client().ContainerRemove(
		Ctx(),
		id,
		container.RemoveOptions{Force: force},
	)
}

// ContainerLogs 获取容器日志（返回流）
func ContainerLogs(id string, opts ContainerLogOptions) (io.ReadCloser, error) {
	return Client().ContainerLogs(
		Ctx(),
		id,
		container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     opts.Follow,
			Tail:       opts.Tail,
		},
	)
}

type ContainerExecOptions struct {
	Workdir string
	Env     []string
	User    string
	Stdin   io.Reader
}

// ExecContainer 在容器中执行命令
func ExecContainer(id string, cmd []string, opts ContainerExecOptions) (string, error) {
	ctx := context.Background()

	exec, err := Client().ContainerExecCreate(ctx, id, types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  opts.Stdin != nil,
		WorkingDir:   opts.Workdir,
		Env:          opts.Env,
		User:         opts.User,
		Tty:          false,
	})
	if err != nil {
		return "", err
	}

	resp, err := Client().ContainerExecAttach(
		ctx,
		exec.ID,
		types.ExecStartCheck{},
	)
	if err != nil {
		return "", err
	}
	defer resp.Close()

	// 1️⃣ stdin → container
	if opts.Stdin != nil {
		go func() {
			_, _ = io.Copy(resp.Conn, opts.Stdin)
			_ = resp.CloseWrite()
		}()
	}

	// Docker's non-TTY exec stream multiplexes stdout and stderr with an
	// 8-byte header. Demultiplex it so callers never receive protocol bytes.
	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, resp.Reader)
	if err != nil {
		return "", err
	}

	inspect, err := Client().ContainerExecInspect(ctx, exec.ID)
	if err != nil {
		return "", err
	}
	if inspect.ExitCode != 0 {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = fmt.Sprintf("container exec exited with code %d", inspect.ExitCode)
		}
		return "", fmt.Errorf("container exec failed (exit %d): %s", inspect.ExitCode, message)
	}

	return stdout.String(), nil
}

type ContainerRunOptions struct {
	container.Config
	container.HostConfig
	network.NetworkingConfig
	ocispec.Platform
	containerName string
}

func NewRunOptions(name, image string) *ContainerRunOptions {
	return &ContainerRunOptions{
		Config: container.Config{
			Image: image,
		},
		HostConfig: container.HostConfig{},
		NetworkingConfig: network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{},
		},
		containerName: name,
	}
}

// ResourceContainerName gives every namespaced resource a globally unique,
// human-readable Docker container name.
func ResourceContainerName(namespace, resourceType string, parts ...string) string {
	nameParts := []string{namespace, resourceType}
	nameParts = append(nameParts, parts...)
	return strings.Join(nameParts, "_")
}

func (o *ContainerRunOptions) WithEnv(key string, value interface{}) {
	strValue := fmt.Sprintf("%v", value)
	o.Env = append(o.Env, key+"="+strValue)
}

// WithPort 绑定端口 host:container（tcp）
func (o *ContainerRunOptions) WithPort(hostPort int, containerPort int) {
	if o.ExposedPorts == nil {
		o.ExposedPorts = nat.PortSet{}
	}
	if o.PortBindings == nil {
		o.PortBindings = nat.PortMap{}
	}

	port := nat.Port(fmt.Sprintf("%d/tcp", containerPort))
	o.ExposedPorts[port] = struct{}{}

	o.PortBindings[port] = append(
		o.PortBindings[port],
		nat.PortBinding{
			HostPort: fmt.Sprintf("%d", hostPort),
		},
	)
}

// WithVolume 绑定卷 /host:/container[:options]
func (o *ContainerRunOptions) WithVolume(
	source string,
	target string,
	options ...string,
) {
	// 1️⃣ bind mount（宿主机路径）
	if strings.HasPrefix(source, "/") {
		bind := source + ":" + target
		if len(options) > 0 {
			bind += ":" + strings.Join(options, ",")
		}
		o.Binds = append(o.Binds, bind)
		return
	}

	// 2️⃣ volume mount（Docker volume）
	m := mount.Mount{
		Type:   mount.TypeVolume,
		Source: source,
		Target: target,
	}

	// 只解析最常见参数，避免复杂化
	for _, opt := range options {
		if opt == "ro" {
			m.ReadOnly = true
		}
	}

	o.Mounts = append(o.Mounts, m)
}

// WithCommand 设置容器启动命令（等价 docker run IMAGE CMD...）
func (o *ContainerRunOptions) WithCommand(cmd ...string) {
	o.Cmd = cmd
}

func (o *ContainerRunOptions) WithLabel(k, v string) {
	if o.Labels == nil {
		o.Labels = map[string]string{}
	}
	o.Labels[k] = v
}

func (o *ContainerRunOptions) WithRestart(mode container.RestartPolicyMode) {
	o.RestartPolicy = container.RestartPolicy{Name: mode}
}

func (o *ContainerRunOptions) WithLogging(driver, maxSize string, maxFile int) {
	if driver == "" {
		return
	}
	config := map[string]string{}
	if maxSize != "" {
		config["max-size"] = maxSize
	}
	if maxFile > 0 {
		config["max-file"] = fmt.Sprintf("%d", maxFile)
	}
	o.LogConfig = container.LogConfig{Type: driver, Config: config}
}

// RecreateContainer applies create-only settings while preserving the image,
// environment, mounts, ports and network attachments of the old container.
func RecreateContainer(id string, restart container.RestartPolicyMode, logConfig container.LogConfig, extraLabels map[string]string) (string, error) {
	inspect, err := InspectContainer(id)
	if err != nil {
		return "", err
	}
	if inspect.Config == nil || inspect.HostConfig == nil {
		return "", fmt.Errorf("container configuration is unavailable")
	}
	name := strings.TrimPrefix(inspect.Name, "/")
	wasRunning := inspect.State != nil && inspect.State.Running
	if wasRunning {
		if err := StopContainer(id, nil); err != nil {
			return "", err
		}
	}
	if err := RemoveContainer(id, false); err != nil {
		return "", err
	}
	hostConfig := *inspect.HostConfig
	hostConfig.RestartPolicy = container.RestartPolicy{Name: restart}
	hostConfig.LogConfig = logConfig
	if inspect.Config.Labels == nil {
		inspect.Config.Labels = map[string]string{}
	}
	for key := range inspect.Config.Labels {
		if strings.HasPrefix(key, "dockflow.sls.") {
			delete(inspect.Config.Labels, key)
		}
	}
	for key, value := range extraLabels {
		inspect.Config.Labels[key] = value
	}
	networking := &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{}}
	if inspect.NetworkSettings != nil {
		for networkName := range inspect.NetworkSettings.Networks {
			networking.EndpointsConfig[networkName] = &network.EndpointSettings{}
		}
	}
	response, err := Client().ContainerCreate(Ctx(), inspect.Config, &hostConfig, networking, nil, name)
	if err != nil {
		return "", err
	}
	if wasRunning {
		if err := StartContainer(response.ID); err != nil {
			return response.ID, err
		}
	}
	return response.ID, nil
}

func (o *ContainerRunOptions) WithNetwork(name string) {
	if o.EndpointsConfig == nil {
		o.EndpointsConfig = map[string]*network.EndpointSettings{}
	}
	o.EndpointsConfig[name] = &network.EndpointSettings{}
}

// 设置 CPU 核心数，例如 0.5 / 1 / 2
func (o *ContainerRunOptions) WithCpu(cpu float64) {
	if cpu <= 0 {
		return
	}
	o.HostConfig.NanoCPUs = int64(cpu * 1e9)
}

// 设置内存大小（单位：GB，例如 0.5 / 1 / 2）
func (o *ContainerRunOptions) WithMemory(memory float64) {
	if memory <= 0 {
		return
	}
	o.HostConfig.Memory = int64(memory * 1024 * 1024 * 1024)
}

func RunContainer(opts *ContainerRunOptions) (string, error) {
	ctx := Ctx()

	resp, err := Client().ContainerCreate(
		ctx,
		&opts.Config,
		&opts.HostConfig,
		&opts.NetworkingConfig,
		&opts.Platform,
		opts.containerName,
	)
	if err != nil {
		return "", err
	}

	if err := Client().ContainerStart(
		ctx,
		resp.ID,
		container.StartOptions{},
	); err != nil {
		return "", err
	}

	return resp.ID, nil
}
