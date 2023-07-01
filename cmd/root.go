package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("goki")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("goki")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Did not find a GoKi configuration file in current directory, so using the default values.\nTo make a configuration file, create a file named goki.toml, goki.yaml, or goki.json in your current directory.")
			fmt.Println()
		} else {
			fmt.Fprintln(os.Stderr, "error loading configuration file:", err)
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   "goki",
	Short: "GoKi is an open source 2D and 3D GUI framework",
	Long:  `GoKi is a free and open source framework for building beautiful, useful, and fast 2D and 3D GUIs for desktop, mobile, and web.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
