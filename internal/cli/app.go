package cli

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(appCmd)
	appCmd.AddCommand(appAddCmd, appListCmd)
}

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage applications",
}

var appAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add application",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		println("[TODO] add app:", args[0])
	},
}

var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List applications",
	Run: func(cmd *cobra.Command, args []string) {
		println("[TODO] list apps")
	},
}
