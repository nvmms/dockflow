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
	databaseCmd.AddCommand(databaseCreateCmd, databaseListCmd, databaseRemoveCmd, databaseExportCmd, databaseImportCmd)

	databaseCreateCmd.Flags().Float64("cpu", 1, "CPU limit (cores)")
	databaseCreateCmd.Flags().Float64("memory", 2, "Memory limit:GB")
	databaseCreateCmd.Flags().String("username", "", "")
	databaseCreateCmd.Flags().String("password", "", "")
	databaseCreateCmd.Flags().String("dbname", "", "")
	databaseCreateCmd.Flags().String("dbtype", "mysql:5.7", "Database type mysql pgsql support")
	databaseCreateCmd.Flags().Bool("remote", false, "open remote access, use random port bind mysql 3306")
}

var (
	ErrorUsernamBlock   = errors.New("username can't be block")
	ErrorPasswordBlock  = errors.New("password can't be block")
	ErrorDbnameBlock    = errors.New("databasename can't be block")
	ErrorNameSpaceBlock = errors.New("Namespace can't be block")
	namespace           = ""
)

var databaseCmd = &cobra.Command{
	Use:     "database",
	Aliases: []string{"db"},
	Short:   "Manage database in the specified namespace",
}

var databaseCreateCmd = &cobra.Command{
	Use:   "create <namespace> <name>",
	Short: "Create database instance",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace := args[0]
		name := args[1]

		cpu, _ := cmd.Flags().GetFloat64("cpu")
		memory, _ := cmd.Flags().GetFloat64("memory")
		username, _ := cmd.Flags().GetString("username")
		if username == "" {
			return ErrorUsernamBlock
		}
		password, _ := cmd.Flags().GetString("password")
		if password == "" {
			return ErrorPasswordBlock
		}
		dbname, _ := cmd.Flags().GetString("dbname")
		if dbname == "" {
			return ErrorDbnameBlock
		}
		dbtype, _ := cmd.Flags().GetString("dbtype")
		remote, _ := cmd.Flags().GetBool("remote")

		database := domain.DatabaseSpec{
			Namespace: namespace,
			Name:      name,
			CPU:       cpu,
			Memory:    memory,
			Username:  username,
			Password:  password,
			DbName:    dbname,
			DbType:    dbtype,
			Remote:    remote,
		}
		err := usecase.Createdatabase(database)
		if err != nil {
			return err
		}
		return nil
	},
}

var databaseListCmd = &cobra.Command{
	Use:     "list <namespace>",
	Short:   "List database",
	Aliases: []string{"ls"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace := args[0]

		dbList, err := usecase.Listdatabase(namespace)
		if err != nil {
			return err
		}
		printDatabaseList(dbList)
		return nil
	},
}

var databaseRemoveCmd = &cobra.Command{
	Use:     "remove <namespace> <name>",
	Short:   "Remove database",
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace := args[0]
		name := args[1]

		err := usecase.Removedatabase(namespace, name)
		if err != nil {
			return err
		}
		return nil
	},
}

func printDatabaseList(list []domain.DatabaseSpec) {
	fmt.Printf("%-12s %-8s %-8s %-10s\n", "NAME", "TYPE", "VERSION", "STATUS")
	fmt.Println("----------------------------------------")

	for _, db := range list {
		fmt.Printf(
			"%-12s %-8s %-8s %-10s\n",
			db.Name,
			db.DbType, // mysql / pg
		)
	}
}

var databaseExportCmd = &cobra.Command{
	Use:   "export <namespace> <name>",
	Short: "export database",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace := args[0]
		name := args[1]
		return usecase.ExportSQL(namespace, name)
	},
}

var databaseImportCmd = &cobra.Command{
	Use:   "import <namespace> <name>",
	Short: "import database",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace := args[0]
		name := args[1]

		return usecase.ImportSQL(namespace, name, "")
	},
}
