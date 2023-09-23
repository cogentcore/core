// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"goki.dev/goki/config"
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

	// Don't use runCmd as adb does not return a useful exit code.
	cmd := exec.Command(
		`adb`,
		`install`,
		`-r`,
		AndroidPkgName(path.Base(pkg.PkgPath))+`.apk`,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if c.Build.Print || c.Build.PrintOnly {
		PrintCmd("%s", strings.Join(cmd.Args, " "))
	}
	if c.Build.PrintOnly {
		return nil
	}
	return cmd.Run()
}
