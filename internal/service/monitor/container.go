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
}

func NewMonitorContainer(containerId string) *MonitorContainer {
	container := &MonitorContainer{
		ContainerId: containerId,
	}

	err := container.findApp()
	if err != nil {
		log.Println("[error]", err)
		return nil
	}
	container.TraefikConfigFile = container.getTraefikConfigFile()
	return container
}

func (m *MonitorContainer) findApp() error {
	containerInfo, err := docker.InspectContainer(m.ContainerId)
	if err != nil {
		return err
	}

	labels := containerInfo.Config.Labels

	namespace, exists := labels["dockflow.namespace"]
	if !exists || namespace == "" {
		return fmt.Errorf("namespace [%s] not set", namespace)
	}

	name, exists := labels["dockflow.name"]
	if !exists || name == "" {
		return fmt.Errorf("app name [%s] not set", name)
	}

	version, exists := labels["dockflow.version"]
	if !exists || version == "" {
		return fmt.Errorf("app version [%s] not set", version)
	}

	ns, err := filesystem.LoadNamespace(namespace)
	if err != nil {
		return err
	}
	if ns == nil {
		return fmt.Errorf("namespace [%s] not found", err)
	}

	app, found := lo.Find(ns.App, func(app domain.AppSpec) bool {
		return app.Name == name
	})
	if !found {
		return fmt.Errorf("app [%s] not found", app.Name)
	}
	if app.Name == "" {
		return fmt.Errorf("app [%s] not found", app.Name)
	}

	deploy, found := lo.Find(app.Deploy, func(deploy domain.AppDeploy) bool {
		return deploy.Version == version
	})
	if !found {
		return fmt.Errorf("deploy version [%s] not found", version)
	}

	m.App = app
	m.ContainerInfo = containerInfo
	m.Deploy = deploy
	return nil
}

func (m *MonitorContainer) onStart() {
	log.Println("[container onStart]", m.ContainerId)

	traefikNetworkIp := ""
	for networkName, network := range m.ContainerInfo.NetworkSettings.Networks {
		if networkName == "dockflow-traefik" {
			traefikNetworkIp = network.IPAddress
		}
	}

	if traefikNetworkIp == "" {
		log.Println("[traefik] container not in dockflow-traefik network")
		return
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
			Name:      m.App.Name + "_" + m.Deploy.Version + "_" + url.Port,
			Rule:      rule,
			Url:       traefikNetworkIp + ":" + url.Port,
			EnableTLS: true,
		}
		cfg.AddService(traefikOpt)
	}

	cfg.Save()
}

func (m *MonitorContainer) onDie() {
	log.Println("[container onDie]", m.ContainerId)
	os.Remove(m.TraefikConfigFile)
	// cfg, err := domain.NewTraefikConfig(m.TraefikConfigFile)
	// if err != nil {
	// 	log.Println("[traefik]      ", err)
	// 	return
	// }

	// for _, url := range m.App.URLs {
	// 	cfg.RemoveService(m.App.Name + "_" + m.Deploy.Version + "_" + url.Port)
	// }

	// cfg.Save()

	// if len(cfg.HTTP.Routers) == 0 && len(cfg.HTTP.Services) == 0 {
	// 	_ = os.Remove(m.TraefikConfigFile)
	// }
}

func (m *MonitorContainer) getTraefikConfigFile() string {
	return filesystem.TraefikCfgDir + "/" + m.App.Name + "_" + m.Deploy.Version + ".yaml"
}
