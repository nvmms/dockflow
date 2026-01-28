package cli

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "dockflow",
	Short: "DockFlow - lightweight docker deployment tool",
}

func Execute() {
	_ = rootCmd.Execute()
}
