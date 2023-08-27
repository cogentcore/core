// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

// func init() {
// 	viper.SetConfigName("config")
// 	viper.AddConfigPath(".goki")
// 	viper.SetEnvPrefix("goki")
// 	viper.AutomaticEnv()
// 	err := viper.ReadInConfig()
// 	if err != nil {
// 		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
// 			// Maybe add back in some way later, but too annoying to have every time.
// 			// fmt.Println("Did not find a GoKi configuration file; use goki init to create one.")
// 			// fmt.Println()
// 		} else {
// 			fmt.Fprintln(os.Stderr, "error loading configuration file:", err)
// 		}
// 	}
// }

// var rootCmd = &cobra.Command{
// 	Use:   "goki",
// 	Short: "GoKi is an open source 2D and 3D GUI framework",
// 	Long:  `GoKi is a free and open source framework for building beautiful, useful, and fast 2D and 3D GUIs for desktop, mobile, and web.`,
// 	RunE: func(cmd *cobra.Command, args []string) error {
// 		return cmd.Help()
// 	},
// }

// func Execute() {
// 	if err := rootCmd.Execute(); err != nil {
// 		os.Exit(1)
// 	}
// }
