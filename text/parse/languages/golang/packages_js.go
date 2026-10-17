// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package golang

// packageGoFile returns the first Go file of the package with the given
// import path. On the web, packages cannot be loaded, since that runs the go
// command, so it always returns false. This keeps golang.org/x/tools/go/packages
// and go/types out of web builds.
func packageGoFile(path string) (string, bool) {
	return "", false
}
