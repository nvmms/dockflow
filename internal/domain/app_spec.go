package domain

import (
	"fmt"
)

type AppURL struct {
	Host string `json:"host"` // external access domain
	Port string `json:"port"` // container port
}

type Env struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Trigger struct {
	Type string `json:"type"` // branch | tag
	Rule string `json:"rule"` // main | v* | v1.*
}

type AppDeploy struct {
	ContainerId string `json:"containerId"`
	Version     string `json:"version"`
	Url         string `json:"url"`
}

type AppSpec struct {
	Namespace string             `json:"namespace"`
	Name      string             `json:"name"`
	CPU       float64            `json:"cpu"`     // CPU cores
	Memory    int                `json:"memory"`  // Memory in GB
	Repo      string             `json:"repo"`    // Git repository
	Token     string             `json:"token"`   // Git access token (optional)
	Trigger   Trigger            `json:"trigger"` // Deploy trigger
	Envs      []Env              `json:"env"`     // Environment variables
	URLs      []AppURL           `json:"url"`     // Access rules
	Deploy    []AppDeploy        `json:"deploy"`
	BuildArg  map[string]*string `json:"buildArg"`
	Secret    string             `json:"secret"`
}

func SaveApp(app AppSpec) error {
	ns, err := NewNamespace(app.Namespace)
	if err != nil {
		return err
	}
	if ns == nil {
		return fmt.Errorf("namespace [%s] not exist", app.Namespace)
	}
	var appIndex = -1
	for index, _app := range ns.App {
		if app.Name == _app.Name {
			appIndex = index
		}
	}
	if appIndex == -1 {
		return fmt.Errorf("app [%s] not exist", app.Name)
	}
	ns.App[appIndex] = app
	ns.Save()
	return nil
}
