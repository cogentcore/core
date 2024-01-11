// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"fmt"
	"path/filepath"

	"goki.dev/grog"
	"goki.dev/xe"
)

// Clean removes object files and cached NDK files downloaded by gomobile init
func Clean() (err error) {
	gopaths := filepath.SplitList(GoEnv("GOPATH"))
	if len(gopaths) == 0 {
		return fmt.Errorf("GOPATH is not set")
	}
	GoMobilePath = filepath.Join(gopaths[0], "pkg/gomobile")
	grog.PrintlnInfo("GOMOBILE=" + GoMobilePath)
	return xe.RemoveAll(GoMobilePath)
}
