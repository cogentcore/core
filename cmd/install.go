package cmd

import (
	"fmt"
	"strings"

	"github.com/goki/packman"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a GoKi package",
	Long:  `Install a local or global GoKi package to your device or mobile emulator`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return packman.InstallLocal("")
		}
		if len(args) > 1 {
			return fmt.Errorf("expected 0 or 1 installation arguments, but got %d", len(args))
		}
		arg := args[0]
		if arg == "." || arg == ".." || strings.Contains(arg, "/") {
			return packman.InstallLocal(arg)
		}
		return packman.Install(arg)
	},
}
