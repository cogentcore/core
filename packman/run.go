// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"fmt"
	"path/filepath"
	"runtime"

	"goki.dev/goki/config"
	"goki.dev/xe"
)

// Run builds and runs the config package. It also displays the logs generated
// by the app. It uses the same config info as build.
//
//gti:add
func Run(c *config.Config) error {
	if len(c.Build.Target) != 1 {
		return fmt.Errorf("expected 1 target platform, but got %d (%v)", len(c.Build.Target), c.Build.Target)
	}
	t := c.Build.Target[0]
	// if no arch is specified, we can assume it is the current arch,
	// as the user is running it (it could be a different arch when testing
	// on an external mobile device, but it is up to the user to specify
	// that arch in that case)
	if t.Arch == "*" {
		t.Arch = runtime.GOARCH
		c.Build.Target[0] = t
	}
	err := Build(c)
	if err != nil {
		return fmt.Errorf("error building app: %w", err)
	}
	switch t.OS {
	case "darwin", "windows", "linux":
		return xe.Verbose().SetBuffer(false).Run(filepath.Join(".", c.Build.Output))
	case "android":
		err := xe.Run("adb", "install", "-r", c.Build.Output)
		if err != nil {
			return fmt.Errorf("error installing app: %w", err)
		}
		// see https://stackoverflow.com/a/25398877
		err = xe.Run("adb", "shell", "monkey", "-p", c.Build.ID, "1")
		if err != nil {
			return fmt.Errorf("error starting app: %w", err)
		}
		return Log(c)
	}
	return nil
}
