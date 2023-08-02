package cmd

import (
	"fmt"

	"github.com/goki/packman"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(colorgenCmd)
}

var colorgenCmd = &cobra.Command{
	Use:   "colorgen filepath",
	Short: "Generate a color scheme declaration file",
	Long:  "Generate a Go color scheme declaration file from a Material Theme Builder Android Views XML file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("colorgen: expected 1 argument (filepath), not %d arguments", len(args))
		}
		return packman.GenerateColorScheme(args[0], "colorschemes.go")
	},
}
