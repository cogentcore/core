// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"fmt"
	"path/filepath"

	"goki.dev/goki/config"
	"goki.dev/xe"
)

// Run builds and runs the config package. It also displays the logs generated
// by the app. It uses the same config info as build.
func Run(c *config.Config) error {
	if len(c.Build.Target) > 0 {
		return fmt.Errorf("can only run on 1 platform at a time, but got %d (%v)", len(c.Build.Target), c.Build.Target)
	}
	err := Build(c)
	if err != nil {
		return fmt.Errorf("error building app: %w", err)
	}
	t := c.Build.Target[0]
	switch t.OS {
	case "darwin", "windows", "linux":
		return xe.Verbose().SetBuffer(false).Run(filepath.Join(".", c.Build.Output))
	case "android":
		err := xe.Run("adb", "install", "-r", c.Build.Output)
		if err != nil {
			return fmt.Errorf("error installing app: %w", err)
		}
		// see https://stackoverflow.com/a/25398877
		return xe.Run("adb", "shell", "monkey", "-p", c.Build.ID, "1")
	}
	return nil
}
