package cli

import (
	"fmt"
	"os"

	"dockflow/internal/domain"
	"dockflow/internal/usecase"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(nsCmd)
	nsCmd.AddCommand(nsCreateCmd, nsListCmd, nsRemoveCmd, nsInspectCmd)
}

var nsCmd = &cobra.Command{
	Use:     "namespace",
	Aliases: []string{"ns"},
	Short:   "Manage namespaces",
}

/* ---------------- create ---------------- */

var nsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create namespace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ns, err := usecase.CreateNamespace(args[0])
		if err != nil {
			return err
		}

		printNamespaces([]domain.Namespace{*ns})
		return nil
	},
}

/* ---------------- list ---------------- */

var nsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List namespaces",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		list := usecase.ListNamespace()
		printNamespaces(list)
	},
}

/* ---------------- remove ---------------- */

var nsRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove namespace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := usecase.RemoveNamespace(args[0]); err != nil {
			return err
		}

		fmt.Printf("Namespace '%s' removed\n", args[0])
		return nil
	},
}

/* ---------------- inspect ---------------- */

var nsInspectCmd = &cobra.Command{
	Use:   "inspect <name>",
	Short: "Inspect namespace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ns, err := usecase.InspectNamespace(args[0])
		if err != nil {
			return err
		}
		if ns == nil {
			printNamespaces(nil)
			return nil
		}
		printNamespaces([]domain.Namespace{*ns})
		return nil
	},
}

/* ================= helpers ================= */

func printError(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err.Error())
}

func printNamespaces(list []domain.Namespace) {
	// 表头：永远输出
	fmt.Printf("%-20s %-25s %-18s %-18s %-12s\n",
		"NAME", "NETWORK", "SUBNET", "GATEWAY", "NETWORK_ID",
	)

	// 表格行：一行一个 namespace
	for _, ns := range list {
		fmt.Printf("%-20s %-25s %-18s %-18s %-12s\n",
			ns.Name,
			ns.Network,
			ns.Subnet,
			ns.Gateway,
			shortID(ns.NetworkId),
		)
	}
}

// Docker 风格：ID 太长时做短显示（12位）
func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}
