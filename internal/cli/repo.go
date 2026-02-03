package cli

import (
	"dockflow/internal/usecase"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var (
	supportGitRepo = []string{"github", "gitee", "gitlab"}
)

func init() {
	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoAddCmd, repoUpdateCmd, repoRemoveCmd, repoListCmd)

	// repoAddCmd.Flags().String("repo", "", fmt.Sprintf("git repo type : %v", supportGitRepo))
	repoAddCmd.Flags().String("url", "", "only gitlab repo need set url")
	repoAddCmd.Flags().String("name", "", "git repo name")
	repoAddCmd.Flags().String("token", "", "git repo token,please select <read repo>、<write:repo_hook> scopes")

	// repoUpdateCmd.Flags().String("repo", "", fmt.Sprintf("git repo type : %v", supportGitRepo))
	repoUpdateCmd.Flags().String("url", "", "only gitlab repo need set url")
	repoUpdateCmd.Flags().String("name", "", "git repo name")
	repoUpdateCmd.Flags().String("token", "", "git repo token,please select <read repo>、<write:repo_hook> scopes")

	// repoRemoveCmd.Flags().String("repo", "", fmt.Sprintf("git repo type : %v", supportGitRepo))
	repoRemoveCmd.Flags().String("url", "", "only gitlab repo need set url")
	repoRemoveCmd.Flags().String("name", "", "git repo name")
}

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage git repo token",
}

var repoAddCmd = &cobra.Command{
	Use:   "add <repo>",
	Short: "add git repo token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := args[0]
		if repo == "" {
			return errors.New("repo can't be blank")
		}
		if !lo.Contains(supportGitRepo, repo) {
			return fmt.Errorf("repo [%s] not support, only support %v", repo, supportGitRepo)
		}

		url, _ := cmd.Flags().GetString("url")
		if repo == "gitlab" && url == "" {
			return fmt.Errorf("repo gitlab must be set url")
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return errors.New("name can't be blank")
		}

		token, _ := cmd.Flags().GetString("token")
		if token == "" {
			return errors.New("token can't be blank")
		}

		return usecase.RepoAdd(map[string]string{
			"url":   url,
			"repo":  repo,
			"name":  name,
			"token": token,
		})
	},
}

var repoUpdateCmd = &cobra.Command{
	Use:   "update <repo>",
	Short: "update git repo token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := args[0]
		if repo == "" {
			return errors.New("repo can't be blank")
		}
		if !lo.Contains(supportGitRepo, repo) {
			return fmt.Errorf("repo [%s] not support, only support %v", repo, supportGitRepo)
		}

		url, _ := cmd.Flags().GetString("url")
		if repo == "gitlab" && url == "" {
			return fmt.Errorf("repo gitlab must be set url")
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return errors.New("name can't be blank")
		}

		token, _ := cmd.Flags().GetString("token")
		if token == "" {
			return errors.New("token can't be blank")
		}

		return usecase.RepoUpdate(map[string]string{
			"url":   url,
			"repo":  repo,
			"name":  name,
			"token": token,
		})
	},
}

var repoRemoveCmd = &cobra.Command{
	Use:     "remove <repo>",
	Short:   "remove git repo token",
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := args[0]
		if repo == "" {
			return errors.New("repo can't be blank")
		}
		if !lo.Contains(supportGitRepo, repo) {
			return fmt.Errorf("repo [%s] not support, only support %v", repo, supportGitRepo)
		}

		url, _ := cmd.Flags().GetString("url")
		if repo == "gitlab" && url == "" {
			return fmt.Errorf("repo gitlab must be set url")
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return errors.New("name can't be blank")
		}

		return usecase.RepoRemove(map[string]string{
			"url":  url,
			"repo": repo,
			"name": name,
		})
	},
}

var repoListCmd = &cobra.Command{
	Use:     "list",
	Short:   "list git repo token",
	Aliases: []string{"ls"},
	Args:    cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		git, err := usecase.RepoList()
		if err != nil {
			return err
		}
		for _, v := range git.Gitee {
			print(v.Name)
			print(v.Token)
		}
		for _, v := range git.Github {
			print(v.Name)
			print(v.Token)
		}
		for _, v := range git.Gitlab {
			print(v.Url)
			print(v.Name)
			print(v.Token)
		}
		return nil
	},
}
