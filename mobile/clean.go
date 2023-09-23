// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"

	"goki.dev/goki/config"
)

// Clean removes object files and cached NDK files downloaded by gomobile init
func Clean(c *config.Config) (err error) {
	gopaths := filepath.SplitList(goEnv("GOPATH"))
	if len(gopaths) == 0 {
		return fmt.Errorf("GOPATH is not set")
	}
	gomobilepath = filepath.Join(gopaths[0], "pkg/gomobile")
	if c.Build.Print {
		fmt.Fprintln(Xout, "GOMOBILE="+gomobilepath)
	}
	return removeAll(gomobilepath)
}
