package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version          string               `yaml:"version"`
	CurrentNamespace string               `yaml:"currentNamespace"`
	Namespaces       map[string]Namespace `yaml:"namespaces"`
	Platform         Platform             `yaml:"platform"`
}

type Namespace struct {
	Apps map[string]App `yaml:"apps"`
}

type App struct {
	Git  string `yaml:"git"`
	Path string `yaml:"path"`
}

type Platform struct {
	Traefik Traefik `yaml:"traefik"`
}

type Traefik struct {
	AcmeEmail   string `yaml:"acmeEmail"`
	ContainerId string `yaml:"containerId"`
}

// Load loads dockflow config from ~/.dockflow/dockflow.yaml
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cfgPath := filepath.Join(home, ".dockflow", "dockflow.yaml")

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Version == "" {
		return nil, errors.New("invalid config: version is empty")
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := filepath.Join(home, ".dockflow", "dockflow.yaml")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
