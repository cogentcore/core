// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import goki "goki.dev/goki/tools"

// InitCmd initializes the ".goki" directory
// and the configuration file in the current directory
func (a *App) InitCmd() error {
	return goki.Init()
}

// func init() {
// 	rootCmd.AddCommand(initCmd)
// }

// var initCmd = &cobra.Command{
// 	Use:   "init",
// 	Short: `Initialize the ".goki" directory`,
// 	Long:  `Initialize the ".goki" directory and the configuration file in the current directory`,
// 	RunE: func(cmd *cobra.Command, args []string) error {
// 		if len(args) > 0 {
// 			return errors.New("expected 0 arguments to init, but got " + strconv.Itoa(len(args)))
// 		}
// 		return goki.Init()
// 	},
// }
