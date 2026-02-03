package monitor

import (
	"dockflow/internal/domain"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/filesystem"
	"fmt"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/samber/lo"
)

type MonitorContainer struct {
	ContainerId       string
	ContainerInfo     types.ContainerJSON
	App               domain.AppSpec
	Deploy            domain.AppDeploy
	TraefikConfigFile string
	Version           string
}

func NewMonitorContainer(containerId string) MonitorContainer {
	container := MonitorContainer{
		ContainerId: containerId,
	}
	return container
}

func (m *MonitorContainer) findApp() {
	containerInfo, err := docker.InspectContainer(m.ContainerId)
	if err != nil {
		fmt.Print(err)
		return
	}

	labels := containerInfo.Config.Labels

	namespace, exists := labels["dockflow.namespace"]
	if !exists || namespace == "" {
		return
		// return nil, &containerInfo, fmt.Errorf("namespace [%s] not set", namespace)
	}

	name, exists := labels["dockflow.name"]
	if !exists || name == "" {
		return
		// return nil, &containerInfo, fmt.Errorf("app name [%s] not set", name)
	}

	version, exists := labels["dockflow.version"]
	if !exists || version == "" {
		return
		// return nil, &containerInfo, fmt.Errorf("app version [%s] not set", version)
	}

	ns, err := filesystem.LoadNamespace(namespace)
	if err != nil {
		return
		// return nil, &containerInfo, err
	}
	if ns == nil {
		return
		// return nil, &containerInfo, fmt.Errorf("namespace [%s] not found", err)
	}

	app, found := lo.Find(ns.App, func(app domain.AppSpec) bool {
		return app.Name == name
	})
	if !found {
		return
		// return nil, &containerInfo, fmt.Errorf("app [%s] not found", app.Name)
	}
	if app.Name == "" {
		return
		// return nil, &containerInfo, fmt.Errorf("app [%s] not found", app.Name)
	}

	deploy, found := lo.Find(app.Deploy, func(deploy domain.AppDeploy) bool {
		return deploy.Version == version
	})
	if !found {
		return
	}

	m.App = app
	m.ContainerInfo = containerInfo
	m.Deploy = deploy
}

func (m *MonitorContainer) onStart() {
	log.Println("[container onStart]", m.ContainerId)
	m.findApp()
	m.TraefikConfigFile = filesystem.TraefikCfgDir + "/" + m.App.Name + ".yaml"

	traefikNetworkIp := ""
	for networkName, network := range m.ContainerInfo.NetworkSettings.Networks {
		if networkName == "dockflow-traefik" {
			traefikNetworkIp = network.IPAddress
		}
	}

	cfg, err := domain.NewTraefikConfig(m.TraefikConfigFile)
	if err != nil {
		log.Println("[traefik]      ", err)
		return
	}

	for _, url := range m.App.URLs {
		rule := url.Host
		if m.Deploy.Version != "latest" {
			rule += "/" + m.Deploy.Version
		}
		traefikOpt := domain.TraefikServiceOpt{
			Name: m.App.Name + "_" + m.Deploy.Version + "_" + url.Port,
			Rule: rule,
			Url:  traefikNetworkIp + ":" + url.Port,
		}
		cfg.AddService(traefikOpt)
	}

	cfg.Save()
}

func (m *MonitorContainer) onDie() {
	log.Println("[container onDie]", m.ContainerId)
	os.Remove(m.TraefikConfigFile)
}
