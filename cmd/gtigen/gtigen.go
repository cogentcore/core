// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/grease"
	"goki.dev/gti/gtigen"
)

func main() {
	grease.AppName = "gtigen"
	grease.AppTitle = "GTIGen"
	grease.AppAbout = "GTIGen provides the generation of general purpose type information for Go types, methods, functions and variables"
	grease.DefaultFiles = []string{"gtigen.toml"}
	grease.Run(&gtigen.Config{}, gtigen.Generate)
}
