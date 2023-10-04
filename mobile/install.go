// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"fmt"

	"goki.dev/goki/config"
	"goki.dev/xe"
)

// Install installs the app named by the import path on the attached mobile device.
// It assumes that it has already been built.
//
// On Android, the 'adb' tool must be on the PATH.
// On iOS, Install also runs the app.
func Install(c *config.Config) error {
	if len(c.Build.Target) != 1 {
		return fmt.Errorf("need at least one target platform for install")
	}
	t := c.Build.Target[0]
	switch t.OS {
	case "android":
		return xe.Run("adb", "install", "-r", c.Build.Output)
	case "ios":
		return xe.Run("ios-deploy", "--debug", "--bundle", c.Build.Output)
	default:
		return fmt.Errorf("mobile.Install only supports target platforms android and ios, but got %q", t.OS)
	}
}
