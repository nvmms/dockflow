package cli

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(nsCmd)
	nsCmd.AddCommand(nsCreateCmd, nsListCmd)
}

var nsCmd = &cobra.Command{
	Use:     "namespace",
	Aliases: []string{"ns"},
	Short:   "Manage namespaces",
}

var nsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create namespace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		println("[TODO] create namespace:", args[0])
	},
}

var nsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List namespaces",
	Run: func(cmd *cobra.Command, args []string) {
		println("[TODO] list namespaces")
	},
}
