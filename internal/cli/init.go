package cli

import (
	"dockflow/internal/service/docker"
	"dockflow/internal/usecase"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize dockflow workspace",
	Run: func(cmd *cobra.Command, args []string) {
		if err := usecase.Init(); err != nil {
			switch err {
			case docker.ErrDockerNotFound:
				println("Docker not found. Please install Docker first.")
			case docker.ErrDockerNotRunning:
				println("Docker daemon is not running.")
			case docker.ErrDockerNoPerm:
				println("No permission to access Docker. Try adding user to docker group.")
			default:
				println("Init failed:", err.Error())
			}
			return
		}
		println("✔ DockFlow environment ready")
	},
}
