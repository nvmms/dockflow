package cli

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy <app>",
	Short: "Deploy application",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		update, _ := cmd.Flags().GetBool("update")
		rollback, _ := cmd.Flags().GetBool("rollback")

		println("[TODO] deploy app:", args[0],
			"update=", update,
			"rollback=", rollback)
	},
}

func init() {
	deployCmd.Flags().Bool("update", false, "Update deployment")
	deployCmd.Flags().Bool("rollback", false, "Rollback deployment")
}
