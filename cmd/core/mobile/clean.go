// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"fmt"
	"path/filepath"

	"cogentcore.org/core/exec"
	"cogentcore.org/core/logx"
)

// Clean removes object files and cached NDK files downloaded by gomobile init
func Clean() (err error) {
	gopaths := filepath.SplitList(GoEnv("GOPATH"))
	if len(gopaths) == 0 {
		return fmt.Errorf("GOPATH is not set")
	}
	GoMobilePath = filepath.Join(gopaths[0], "pkg/gomobile")
	logx.PrintlnInfo("GOMOBILE=" + GoMobilePath)
	return exec.RemoveAll(GoMobilePath)
}
