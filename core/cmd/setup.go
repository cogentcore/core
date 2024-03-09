// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"runtime"

	"cogentcore.org/core/core/config"
	"cogentcore.org/core/xe"
)

// Setup installs platform-specific dependencies for the current platform.
// It only needs to be called once per system.
func Setup(c *config.Config) error { //gti:add
	vc := xe.Verbose().SetBuffer(false)
	switch runtime.GOOS {
	case "darwin":
		p, err := xe.Output("xcode-select", "-p")
		if err != nil || p == "" {
			err := vc.Run("xcode-select", "--install")
			if err != nil {
				return err
			}
		}
		err = vc.Run("curl", "-O", "https://sdk.lunarg.com/sdk/download/1.3.261.1/mac/vulkansdk-macos-1.3.261.1.dmg")
		if err != nil {
			return err
		}
		err = vc.Run("open", "vulkansdk-macos-1.3.261.1.dmg")
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("platform %q not supported for core setup", runtime.GOOS)
}
