package cli

import (
	"dockflow/internal/domain"
	"dockflow/internal/usecase"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(databaseCmd)
	databaseCmd.AddCommand(databaseCreateCmd, databaseListCmd, databaseRemoveCmd)

	databaseCreateCmd.Flags().Float64("cpu", 1, "CPU limit (cores)")
	databaseCreateCmd.Flags().Float64("memory", 2, "Memory limit:GB")
	databaseCreateCmd.Flags().String("dbtype", "mysql:5.7", "Database type mysql pgsql support")
	databaseCreateCmd.Flags().String("username", "", "")
	databaseCreateCmd.Flags().String("password", "", "")
	databaseCreateCmd.Flags().String("dbname", "", "")
}

var databaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Manage database",
}

var (
	ErrorUsernamBlock  = errors.New("username can't be block")
	ErrorPasswordBlock = errors.New("password can't be block")
	ErrorDbnameBlock   = errors.New("databasename can't be block")
)

var databaseCreateCmd = &cobra.Command{
	Use:   "create <namespace> <name>",
	Short: "Create redis instance",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		namespace := args[0]
		name := args[1]

		cpu, _ := cmd.Flags().GetFloat64("cpu")
		memory, _ := cmd.Flags().GetFloat64("memory")
		username, _ := cmd.Flags().GetString("username")
		if username == "" {
			printError(ErrorUsernamBlock)
			return
		}
		password, _ := cmd.Flags().GetString("password")
		if password == "" {
			printError(ErrorPasswordBlock)
			return
		}
		dbname, _ := cmd.Flags().GetString("dbname")
		if dbname == "" {
			printError(ErrorDbnameBlock)
			return
		}
		dbtype, _ := cmd.Flags().GetString("dbtype")

		database := domain.DatabaseSpec{
			Namespace: namespace,
			Name:      name,
			CPU:       cpu,
			Memory:    memory,
			Username:  username,
			Password:  password,
			DbName:    dbname,
			DbType:    dbtype,
		}
		err := usecase.Createdatabase(database)
		if err != nil {
			printError(err)
		}

		// aof, _ := cmd.Flags().GetBool("appendonly")
		// policy, _ := cmd.Flags().GetString("maxmemory-policy")

		// redisSpec := domain.NewRedisSpace(name, namespace, password, cpu, memory, version, aof, policy)

		// err := usecase.CreateRedis(redisSpec)
		// if err != nil {
		// 	printError(err)
		// }
	},
}

var databaseListCmd = &cobra.Command{
	Use:     "list <namespace>",
	Short:   "List <namespace> redis",
	Aliases: []string{"ls"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		namespace := args[0]
		dbList, err := usecase.Listdatabase(namespace)
		if err != nil {
			printError(err)
		}
		printDatabaseList(dbList)

	},
}

var databaseRemoveCmd = &cobra.Command{
	Use:     "Remove <namespace> <name>",
	Short:   "Remove <namespace> database",
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		namespace := args[0]
		redisName := args[1]
		err := usecase.Removedatabase(namespace, redisName)
		if err != nil {
			printError(err)
		}
	},
}

func printDatabaseList(list []domain.DatabaseSpec) {
	fmt.Printf("%-12s %-8s %-8s %-10s\n", "NAME", "TYPE", "VERSION", "STATUS")
	fmt.Println("----------------------------------------")

	for _, db := range list {
		fmt.Printf(
			"%-12s %-8s %-8s %-10s\n",
			db.Name,
			db.DbType, // redis / mysql / pg
		)
	}
}
