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
	rootCmd.AddCommand(appCmd)
	appCmd.AddCommand(appCreateCmd, appListCmd, appRemoveCmd, appDeployCmd, appLogCmd, appStatusCmd)

	appCreateCmd.Flags().Float64("cpu", 1, "CPU limit (cores)")
	appCreateCmd.Flags().Int("memory", 1, "Memory limit (GB)")
	appCreateCmd.Flags().String("repo", "", "Git repository url")
	appCreateCmd.Flags().String("token", "", "Git access token")
	// appCreateCmd.Flags().String("platform", "", "deploy platform")
	// appCreateCmd.Flags().String("build-args", "", "build args for platform, json {\"NODE_VERSION\":\"20\",\"BUILD_CMD\":\"npm run build\",\"DIST_DIR\":\"./dist1\"}")
	appCreateCmd.Flags().String(
		"trigger-type",
		"branch",
		"Trigger type: branch or tag",
	)

	appCreateCmd.Flags().String(
		"trigger-rule",
		"main",
		"Trigger rule: branch name or tag pattern",
	)

	// env：key=value，可传多次
	appCreateCmd.Flags().StringArray(
		"env",
		[]string{},
		"Environment variable, format: KEY=VALUE",
	)

	appCreateCmd.Flags().StringArray(
		"url",
		[]string{},
		"app url, format: host:containerPort",
	)

	appDeployCmd.Flags().String("branch", "", "")
	appDeployCmd.Flags().String("commit", "", "")
	appDeployCmd.Flags().String("tag", "", "")
}

var (
	ErrRepoCantBlank  = errors.New("app repo url can't be blank")
	ErrTokenCantBlank = errors.New("app repo token can't be blank")
	ErrUrlCantBlank   = errors.New("app external url can't be blank")
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage app",
}

var appCreateCmd = &cobra.Command{
	Use:   "create <namespace> <name>",
	Short: "Create app instance",
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

		// buildArgsStr, _ := cmd.Flags().GetString("build-args")
		// var buildArgsMap map[string]*string
		// err := json.Unmarshal([]byte(buildArgsStr), &buildArgsMap)
		// if err != nil {
		// 	return fmt.Errorf("build args must be json")
		// }

		// ---------- basic validate ----------
		if repo == "" {
			return fmt.Errorf("repo is required")
		}
		// if token == "" {
		// 	return fmt.Errorf("token is required")
		// }

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

		urls := make([]domain.AppURL, 0, len(urlFlags))
		for _, item := range urlFlags {
			parts := strings.Split(item, ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid url format: %s (expect host:port)", item)
			}
			urls = append(urls, domain.AppURL{
				Host: parts[0],
				Port: parts[1],
			})
		}

		// ---------- ServiceSpec ----------
		spec := domain.AppSpec{
			Namespace: namespace,
			Name:      name,
			CPU:       cpu,
			Memory:    memory,
			Repo:      repo,
			Token:     token,
			Trigger:   trigger,
			Envs:      envs,
			URLs:      urls,
			// BuildArg:  buildArgsMap,
			// Platform:  platform,
		}

		err := usecase.CreateApp(spec)
		if err != nil {
			return err
		}

		return nil
	},
}

var appListCmd = &cobra.Command{
	Use:     "list <namespace>",
	Short:   "list app instance",
	Aliases: []string{"ls"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace := args[0]
		apps, err := usecase.ListApp(namespace)
		if err != nil {
			return err
		}
		for _, app := range apps {
			fmt.Printf("%s\n", app.Name)
		}
		return nil
	},
}

var appDeployCmd = &cobra.Command{
	Use:   "deploy <namespace> <name>",
	Short: "deploy app instance",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace := args[0]
		name := args[1]

		branch, _ := cmd.Flags().GetString("branch")
		commit, _ := cmd.Flags().GetString("commit")
		tag, _ := cmd.Flags().GetString("tag")

		opt := usecase.DeployAppOptions{
			Namespace: namespace,
			Name:      name,
			Branch:    branch,
			Commit:    commit,
			Tag:       tag,
		}

		err := usecase.DeployApp(opt)
		if err != nil {
			return err
		}

		return nil
	},
}
var appLogCmd = &cobra.Command{}
var appStatusCmd = &cobra.Command{}

var appRemoveCmd = &cobra.Command{
	Use:     "remove <namespace> <name>",
	Short:   "remove app instance",
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace := args[0]
		app := args[1]
		err := usecase.RemoveApp(namespace, app)
		if err != nil {
			return err
		}
		return nil
	},
}
