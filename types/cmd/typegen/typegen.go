// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command typgen provides the generation of type information for
// Go types, methods, and functions.
package main

import (
	"cogentcore.org/core/cli"
	"cogentcore.org/core/types/typegen"
)

func main() {
	opts := cli.DefaultOptions("typegen", "Typegen provides the generation of type information for Go types, methods, and functions.")
	cli.Run(opts, &typegen.Config{}, typegen.Generate)
}
