package cli

import (
	"dockflow/internal/domain"
	"dockflow/internal/usecase"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceCreateCmd, serviceListCmd, serviceRemoveCmd, serviceDeployCmd, serviceLogCmd, serviceStatusCmd)

	serviceCreateCmd.Flags().Float64("cpu", 1, "CPU limit (cores)")
	serviceCreateCmd.Flags().Int("memory", 1, "Memory limit (GB)")
	serviceCreateCmd.Flags().String("repo", "", "Git repository url")
	serviceCreateCmd.Flags().String("token", "", "Git access token")
	serviceCreateCmd.Flags().String(
		"trigger-type",
		"branch",
		"Trigger type: branch or tag",
	)

	serviceCreateCmd.Flags().String(
		"trigger-rule",
		"main",
		"Trigger rule: branch name or tag pattern",
	)

	// env：key=value，可传多次
	serviceCreateCmd.Flags().StringArray(
		"env",
		[]string{},
		"Environment variable, format: KEY=VALUE",
	)

	serviceCreateCmd.Flags().StringArray(
		"url",
		[]string{},
		"Service url, format: host:containerPort",
	)
}

var (
	ErrRepoCantBlank  = errors.New("service repo url can't be blank")
	ErrTokenCantBlank = errors.New("service repo token can't be blank")
	ErrUrlCantBlank   = errors.New("service external url can't be blank")
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage service",
}

var serviceCreateCmd = &cobra.Command{
	Use:   "create <namespace> <name>",
	Short: "Create Service instance",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace := args[0]
		name := args[1]

		cpu, _ := cmd.Flags().GetFloat64("cpu")
		memory, _ := cmd.Flags().GetInt("memory")
		repo, _ := cmd.Flags().GetString("repo")
		token, _ := cmd.Flags().GetString("token")

		triggerType, _ := cmd.Flags().GetString("trigger-type")
		triggerRule, _ := cmd.Flags().GetString("trigger-rule")

		envFlags, _ := cmd.Flags().GetStringArray("env")
		urlFlags, _ := cmd.Flags().GetStringArray("url")

		// ---------- basic validate ----------
		if repo == "" {
			return fmt.Errorf("repo is required")
		}
		if token == "" {
			return fmt.Errorf("token is required")
		}

		// ---------- trigger ----------
		if triggerRule == "" {
			return fmt.Errorf("trigger-rule is required")
		}

		if triggerType != "branch" && triggerType != "tag" {
			return fmt.Errorf("invalid trigger-type: %s", triggerType)
		}

		trigger := domain.Trigger{
			Type: triggerType,
			Rule: triggerRule,
		}

		// ---------- env ----------
		envs := make([]domain.Env, 0, len(envFlags))
		for _, item := range envFlags {
			parts := strings.SplitN(item, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid env format: %s (expect KEY=VALUE)", item)
			}
			envs = append(envs, domain.Env{
				Key:   parts[0],
				Value: parts[1],
			})
		}

		// ---------- url ----------
		if len(urlFlags) == 0 {
			return fmt.Errorf("at least one --url is required")
		}

		urls := make([]domain.ServiceURL, 0, len(urlFlags))
		for _, item := range urlFlags {
			parts := strings.Split(item, ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid url format: %s (expect host:port)", item)
			}
			urls = append(urls, domain.ServiceURL{
				Host: parts[0],
				Port: parts[1],
			})
		}

		// ---------- ServiceSpec ----------
		spec := domain.ServiceSpec{
			Namespace: namespace,
			Name:      name,
			CPU:       cpu,
			Memory:    memory,
			Repo:      repo,
			Token:     token,
			Trigger:   trigger,
			Envs:      envs,
			URLs:      urls,
		}

		err := usecase.CreateService(spec)
		if err != nil {
			return err
		}

		return nil
	},
}
var serviceListCmd = &cobra.Command{}
var serviceRemoveCmd = &cobra.Command{}
var serviceDeployCmd = &cobra.Command{}
var serviceLogCmd = &cobra.Command{}
var serviceStatusCmd = &cobra.Command{}
