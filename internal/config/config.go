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
}

type Namespace struct {
	Apps map[string]App `yaml:"apps"`
}

type App struct {
	Git  string `yaml:"git"`
	Path string `yaml:"path"`
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
