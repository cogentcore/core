// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/cli"
	"cogentcore.org/core/gti/gtigen"
)

func main() {
	opts := cli.DefaultOptions("gtigen", "GTIGen", "GTIGen provides the generation of general purpose type information for Go types, methods, functions and variables")
	cli.Run(opts, &gtigen.Config{}, gtigen.Generate)
}
