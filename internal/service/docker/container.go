package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type ContainerLogOptions struct {
	Follow bool
	Tail   string
}

type ContainerExecOptions struct {
	Workdir string
	Env     []string
}

// ListContainers 列出容器
func ListContainers(all bool) ([]types.Container, error) {
	return Client().ContainerList(
		Ctx(),
		container.ListOptions{All: all},
	)
}

func HasContainer(containerId string) (bool, error) {
	list, err := ListContainers(true)
	if err != nil {
		return false, err
	}
	for _, item := range list {
		if item.ID == containerId {
			return true, nil
		}
	}
	return false, nil
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

// ExecContainer 在容器中执行命令
func ExecContainer(id string, cmd []string, opts ContainerExecOptions) (string, error) {
	ctx := context.Background()

	exec, err := Client().ContainerExecCreate(ctx, id, types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   opts.Workdir,
		Env:          opts.Env,
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

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Reader)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
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
	hostVolume string,
	containerVolume string,
	options ...string,
) {
	if o.Binds == nil {
		o.Binds = []string{}
	}

	bind := hostVolume + ":" + containerVolume

	if len(options) > 0 {
		bind = bind + ":" + strings.Join(options, ",")
	}

	o.Binds = append(o.Binds, bind)
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

func (o *ContainerRunOptions) WithNetwork(name string) {
	if o.EndpointsConfig == nil {
		o.EndpointsConfig = map[string]*network.EndpointSettings{}
	}
	o.EndpointsConfig[name] = &network.EndpointSettings{}
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
