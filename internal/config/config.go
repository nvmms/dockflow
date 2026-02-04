package config

import (
	"dockflow/internal/service/filesystem"
	"errors"
	"os"

	"github.com/samber/lo"
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
	// AcmeEmail   string `yaml:"acmeEmail"`
	ContainerId string `yaml:"containerId"`
	NetworkId   string `yaml:"networkId"`
}

type GitToken struct {
	Name  string `yaml:"name"`
	Token string `yaml:"token"`
}

type GitGitlab struct {
	Url string `yaml:"url"`
	GitToken
}

type Git struct {
	Gitee  []GitToken  `yaml:"gitee"`
	Github []GitToken  `yaml:"github"`
	Gitlab []GitGitlab `yaml:"gitlab"`
}

// func NewGitConfig(repo map[string]string) (Git, error) {
// 	git := Git{}
// 	switch repo["repo"] {
// 	case "github":
// 		git.Github = append(git.Github, GitToken{
// 			Name:  repo["name"],
// 			Token: repo["token"],
// 		})
// 	case "gitee":
// 		git.Gitee = append(git.Gitee, GitToken{
// 			Name:  repo["name"],
// 			Token: repo["token"],
// 		})
// 	case "gitlab":
// 		git.Gitlab = append(git.Gitlab, GitGitlab{
// 			Url: repo["url"],
// 			GitToken: GitToken{
// 				Name:  repo["name"],
// 				Token: repo["token"],
// 			},
// 		})
// 	default:
// 		return git, fmt.Errorf("repo [%s] not support", repo)
// 	}
// 	return git, nil
// }

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

func FindGit(host string, username string) (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}
	switch host {
	case "gitee.com":
		gitee, found := lo.Find(cfg.Git.Gitee, func(gitee GitToken) bool {
			return gitee.Name == username
		})
		if found {
			return gitee.Token, nil
		}
	case "github.com":
		github, found := lo.Find(cfg.Git.Github, func(github GitToken) bool {
			return github.Name == username
		})
		if found {
			return github.Token, nil
		}
	default:
		gitlab, found := lo.Find(cfg.Git.Gitlab, func(gitlab GitGitlab) bool {
			return gitlab.Name == username && gitlab.Url == host
		})
		if found {
			return gitlab.Token, nil
		}
	}
	return "", nil

}
