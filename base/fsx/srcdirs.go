// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package fsx

import "go/build"

// GoSrcDirs returns the Go source directories: GOROOT/src and GOPATH/src for
// each GOPATH entry. There are none on the web, which also keeps go/build
// and the go/* parsing packages it needs out of web builds.
func GoSrcDirs() []string {
	return build.Default.SrcDirs()
}
