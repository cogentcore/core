package cmd

import (
	"fmt"
	"runtime"

	"github.com/goki/packman"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var buildTarget []string

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.PersistentFlags().StringSliceVarP(&buildTarget, "target", "t", []string{runtime.GOOS + "/" + runtime.GOARCH}, "the target platforms to build executables for, in os[/arch] format")
	viper.BindPFlag("buildTarget", buildCmd.PersistentFlags().Lookup("target"))
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a local package",
	Long:  `Build executables for a local package for one or more platforms`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgPath := ""
		if len(args) != 0 {
			pkgPath = args[0]
		}
		platforms := make([]packman.Platform, len(buildTarget))
		for i, target := range buildTarget {
			platform, err := packman.ParsePlatform(target)
			if err != nil {
				return fmt.Errorf("error parsing build targets: %w", err)
			}
			platforms[i] = platform
		}
		return packman.Build(pkgPath, platforms...)
	},
}
