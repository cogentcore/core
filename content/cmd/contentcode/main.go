// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command contentcode compiles the runnable Go code blocks (```Go) of the
// markdown files of a content app into the app (see
// [cogentcore.org/core/content/contentcode]).
//
// The core tool already does this for every content app in core build, run,
// install, pack and generate; this command is for running it from
// go generate instead, with a directive in the app's main package:
//
//	//go:generate go run cogentcore.org/core/content/cmd/contentcode
//
// The core tool then uses the arguments of the directive.
//
// Usage:
//
//	contentcode [-content content] [-o contentcode_gen.go] [-tags tags] [-import name=path]... [-force]
package main

import (
	"fmt"
	"os"

	"cogentcore.org/core/content/contentcode"
)

func main() {
	c, err := contentcode.ParseArgs(os.Args[0], os.Args[1:])
	if err != nil {
		os.Exit(2)
	}
	if _, err := contentcode.Generate(c); err != nil {
		fmt.Fprintln(os.Stderr, "contentcode:", err)
		os.Exit(1)
	}
}
