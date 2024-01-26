// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package main provides the actual command line
// implementation of the enumgen library.
package main

import (
	"cogentcore.org/core/enums/enumgen"
	"cogentcore.org/core/grease"
)

func main() {
	opts := grease.DefaultOptions("Enumgen", "Enumgen generates helpful methods for Go enums.")
	opts.DefaultFiles = []string{"enumgen.toml"}
	grease.Run(opts, &enumgen.Config{}, enumgen.Generate)
}
