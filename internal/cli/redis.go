package cli

import (
	"dockflow/internal/domain"
	"dockflow/internal/usecase"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(redisCmd)
	redisCmd.AddCommand(redisCreateCmd, redisListCmd, redisRemoveCmd)

	redisCreateCmd.Flags().Float64("cpu", 0.5, "CPU limit (cores)")
	redisCreateCmd.Flags().Float64("memory", 0.5, "Memory limit:GB")
	redisCreateCmd.Flags().String("password", "", "Redis password")
	redisCreateCmd.Flags().String("version", "7", "Redis version")
	redisCreateCmd.Flags().Bool("appendonly", true, "Enable AOF")
	redisCreateCmd.Flags().String("maxmemory-policy", "allkeys-lru", "Eviction policy")
}

var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Manage redis",
}

var redisCreateCmd = &cobra.Command{
	Use:   "create <namespace> <name>",
	Short: "Create redis instance",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		namespace := args[0]
		name := args[1]

		cpu, _ := cmd.Flags().GetFloat64("cpu")
		memory, _ := cmd.Flags().GetFloat64("memory")
		password, _ := cmd.Flags().GetString("password")
		version, _ := cmd.Flags().GetString("version")
		aof, _ := cmd.Flags().GetBool("appendonly")
		policy, _ := cmd.Flags().GetString("maxmemory-policy")

		redisSpec := domain.NewRedisSpace(name, namespace, password, cpu, memory, version, aof, policy)

		err := usecase.CreateRedis(redisSpec)
		if err != nil {
			printError(err)
		}
	},
}

var redisListCmd = &cobra.Command{
	Use:     "list <namespace>",
	Short:   "List <namespace> redis",
	Aliases: []string{"ls"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		namespace := args[0]
		redisList, err := usecase.ListRedis(namespace)
		if err != nil {
			printError(err)
		}
		printRedisList(redisList)
	},
}

var redisRemoveCmd = &cobra.Command{
	Use:     "Remove <namespace> <name>",
	Short:   "Remove <namespace> redis",
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		namespace := args[0]
		redisName := args[1]
		err := usecase.RemoveRedis(namespace, redisName)
		if err != nil {
			printError(err)
		}
	},
}

func printRedisList(list []domain.RedisSpec) {
	fmt.Printf("%-12s %-8s %-8s %-10s\n", "NAME", "TYPE", "VERSION", "STATUS")
	fmt.Println("----------------------------------------")

	for _, db := range list {
		fmt.Printf(
			"%-12s %-8s %-8s %-10s\n",
			db.Name,
			db.Version, // redis / mysql / pg
		)
	}
}
