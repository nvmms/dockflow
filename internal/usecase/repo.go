package usecase

import (
	"dockflow/internal/config"
	"fmt"
	"strings"

	"github.com/samber/lo"
)

func RepoAdd(repo map[string]string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch repo["repo"] {
	case "github":
		for _, github := range cfg.Git.Github {
			if github.Name == repo["name"] {
				return fmt.Errorf("repo github name [%s] is exist, if you need update token, please use [update] command", repo["name"])
			}
		}
		cfg.Git.Github = append(cfg.Git.Github, config.GitToken{
			Name:  repo["name"],
			Token: repo["token"],
		})
	case "gitee":
		for _, gitee := range cfg.Git.Gitee {
			if gitee.Name == repo["name"] {
				return fmt.Errorf("repo gitee name [%s] is exist, if you need update token, please use [update] command", repo["name"])
			}
		}
		cfg.Git.Gitee = append(cfg.Git.Gitee, config.GitToken{
			Name:  repo["name"],
			Token: repo["token"],
		})
	case "gitlab":
		url := getGitlabHost(repo["url"])
		for _, gitlab := range cfg.Git.Gitlab {
			if gitlab.Name == repo["name"] && gitlab.Url == url {
				return fmt.Errorf("repo gitlab [%s] name [%s] is exist, if you need update token, please use [update] command", repo["url"], repo["name"])
			}
		}

		cfg.Git.Gitlab = append(cfg.Git.Gitlab, config.GitGitlab{
			Url: url,
			GitToken: config.GitToken{
				Name:  repo["name"],
				Token: repo["token"],
			},
		})
	default:
		return fmt.Errorf("repo [%s] not support", repo)
	}

	return config.Save(cfg)
}

func RepoUpdate(repo map[string]string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch repo["repo"] {
	case "github":
		_, index, found := lo.FindIndexOf(cfg.Git.Github, func(github config.GitToken) bool {
			return github.Name == repo["name"]
		})
		if !found {
			return fmt.Errorf("repo github name [%s] is not exist", repo["name"])
		}
		cfg.Git.Github[index] = config.GitToken{
			Name:  repo["name"],
			Token: repo["token"],
		}
	case "gitee":
		_, index, found := lo.FindIndexOf(cfg.Git.Gitee, func(gitee config.GitToken) bool {
			return gitee.Name == repo["name"]
		})
		if !found {
			return fmt.Errorf("repo gitee name [%s] is not exist", repo["name"])
		}
		cfg.Git.Gitee[index] = config.GitToken{
			Name:  repo["name"],
			Token: repo["token"],
		}
	case "gitlab":
		url := getGitlabHost(repo["url"])
		_, index, found := lo.FindIndexOf(cfg.Git.Gitlab, func(gitlab config.GitGitlab) bool {
			return gitlab.Name == repo["name"] && gitlab.Url == url
		})
		if !found {
			return fmt.Errorf("repo gitlab url [%s] name [%s] is not exist", url, repo["name"])
		}

		cfg.Git.Gitlab[index] = config.GitGitlab{
			Url: url,
			GitToken: config.GitToken{
				Name:  repo["name"],
				Token: repo["token"],
			},
		}
	default:
		return fmt.Errorf("repo [%s] not support", repo)
	}

	return config.Save(cfg)
}

func RepoRemove(repo map[string]string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch repo["repo"] {
	case "github":
		cfg.Git.Github = lo.Filter(cfg.Git.Github, func(github config.GitToken, index int) bool {
			return github.Name != repo["name"]
		})
	case "gitee":
		cfg.Git.Gitee = lo.Filter(cfg.Git.Gitee, func(gitee config.GitToken, index int) bool {
			return gitee.Name != repo["name"]
		})
	case "gitlab":
		url := getGitlabHost(repo["url"])
		cfg.Git.Gitlab = lo.Filter(cfg.Git.Gitlab, func(gitlab config.GitGitlab, index int) bool {
			return !(gitlab.Name == repo["name"] && gitlab.Url == url)
		})
	default:
		return fmt.Errorf("repo [%s] not support", repo)
	}

	return config.Save(cfg)
}

func getGitlabHost(url string) string {
	url = strings.ReplaceAll(url, "http://", "")
	url = strings.ReplaceAll(url, "https://", "")
	return url
}

func RepoList() (*config.Git, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return &cfg.Git, nil
}
