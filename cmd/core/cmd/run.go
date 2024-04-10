// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"path/filepath"
	"runtime"

	"cogentcore.org/core/cmd/core/config"
	"cogentcore.org/core/cmd/core/mobile"
	"cogentcore.org/core/cmd/core/web"
	"cogentcore.org/core/xe"
	"cogentcore.org/core/xlog"
)

// Run builds and runs the config package. It also displays the logs generated
// by the app. It uses the same config info as build.
func Run(c *config.Config) error { //gti:add
	if len(c.Build.Target) != 1 {
		return fmt.Errorf("expected 1 target platform, but got %d (%v)", len(c.Build.Target), c.Build.Target)
	}
	t := c.Build.Target[0]
	if t.Arch == "*" {
		if t.OS == "android" || t.OS == "ios" {
			t.Arch = "arm64"
		} else {
			t.Arch = runtime.GOARCH
		}
		c.Build.Target[0] = t
	}

	if t.OS == "ios" && !c.Build.Debug {
		// TODO: is there a way to launch without running the debugger?
		xlog.PrintlnWarn("warning: only installing, not running, because there is no effective way to just launch an app on iOS from the terminal without debugging; pass the -d flag to run and debug")
	}

	if t.OS == "web" {
		// needed for changes to show during local development
		c.Web.RandomVersion = true
	}

	err := Build(c)
	if err != nil {
		return fmt.Errorf("error building app: %w", err)
	}
	// Build may have added iossimulator, so we get rid of it for
	// the running stage (we already confirmed we were passed 1 up above)
	if len(c.Build.Target) > 1 {
		c.Build.Target = []config.Platform{t}
	}
	switch t.OS {
	case "darwin", "windows", "linux":
		return xe.Verbose().SetBuffer(false).Run("." + string(filepath.Separator) + c.Name)
	case "android":
		err := xe.Run("adb", "install", "-r", filepath.Join("bin", "android", c.Name+".apk"))
		if err != nil {
			return fmt.Errorf("error installing app: %w", err)
		}
		// see https://stackoverflow.com/a/4567928
		args := []string{"shell", "am", "start", "-n", c.ID + "/org.golang.app.GoNativeActivity"}
		// TODO: get adb am debug on Android working
		// if c.Build.Debug {
		// args = append(args, "-D")
		// }
		err = xe.Run("adb", args...)
		if err != nil {
			return fmt.Errorf("error starting app: %w", err)
		}
		if c.Build.Debug {
			return Log(c)
		}
		return nil
	case "ios":
		if !c.Build.Debug {
			return mobile.Install(c)
		}
		return xe.Verbose().SetBuffer(false).Run("ios-deploy", "-b", filepath.Join("bin", "ios", c.Name+".app"), "-d")
	case "web":
		return web.Serve(c)
	}
	return nil
}
