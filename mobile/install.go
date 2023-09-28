// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"fmt"
	"path"

	"goki.dev/goki/config"
	"goki.dev/xe"
)

// Install compiles and installs the app named by the import path on the
// attached mobile device.
//
// Only -target android is supported. The 'adb' tool must be on the PATH.
func Install(c *config.Config) error {
	// TODO: use install config fields, not build ones
	if len(c.Build.Target) != 1 || c.Build.Target[0].OS != "android" {
		return fmt.Errorf("target for install must be android, but got %v", c.Build.Target)
	}
	pkg, err := BuildImpl(c)
	if err != nil {
		return err
	}

	return xe.Run("adb", "install", "-r", AndroidPkgName(path.Base(pkg.PkgPath))+".apk")
}
