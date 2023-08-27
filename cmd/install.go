// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"strings"

	"goki.dev/goki/packman"
)

// InstallCmd installs a local or global GoKi
// package to your device or mobile emulator
func (a *App) InstallCmd() error {
	if a.Install.Package == "." || a.Install.Package == ".." || strings.Contains(a.Install.Package, "/") {
		return packman.InstallLocal(&a.Config)
	}
	return packman.Install(&a.Config)
}

// var installTarget []string

// func init() {
// 	rootCmd.AddCommand(installCmd)
// 	installCmd.PersistentFlags().StringSliceVarP(&installTarget, "target", "t", []string{"android"}, "the target platforms to install the executables on, as a list of operating systems (this should include no more than the operating system you are on, android, and ios)")
// 	viper.BindPFlag("installTarget", installCmd.PersistentFlags().Lookup("target"))
// }

// var installCmd = &cobra.Command{
// 	Use:   "install",
// 	Short: "Install a GoKi package",
// 	Long:  `Install a local or global GoKi package to your device or mobile emulator`,
// 	RunE: func(cmd *cobra.Command, args []string) error {
// 		if len(args) == 0 {
// 			return packman.InstallLocal("", installTarget...)
// 		}
// 		if len(args) > 1 {
// 			return fmt.Errorf("expected 0 or 1 installation arguments, but got %d", len(args))
// 		}
// 		arg := args[0]
// 		if arg == "." || arg == ".." || strings.Contains(arg, "/") {
// 			return packman.InstallLocal(arg, installTarget...)
// 		}
// 		return packman.Install(arg)
// 	},
// }
