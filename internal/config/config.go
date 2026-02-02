package config

import (
	"dockflow/internal/service/filesystem"
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version  string   `yaml:"version"`
	Platform Platform `yaml:"platform"`
	Git      Git      `yaml:"git"`
}

type Platform struct {
	Traefik Traefik `yaml:"traefik"`
}

type Traefik struct {
	AcmeEmail   string `yaml:"acmeEmail"`
	ContainerId string `yaml:"containerId"`
	NetworkId   string `yaml:"networkId"`
}

type GitToken struct {
	Token string `yaml:"token"`
}

type GitGitlab struct {
	Url   string `yaml:"url"`
	Token string `yaml:"token"`
}

type Git struct {
	Gitee  GitToken    `yaml:"gitee"`
	Github GitToken    `yaml:"github"`
	Gitlab []GitGitlab `yaml:"gitlab"`
}

func Load() (*Config, error) {
	data, err := os.ReadFile(filesystem.CfgPath)
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
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filesystem.CfgPath, data, 0644)
}
