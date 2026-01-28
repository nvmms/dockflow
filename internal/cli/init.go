package cli

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize dockflow workspace",
	Run: func(cmd *cobra.Command, args []string) {
		println("[TODO] init workspace")
	},
}
