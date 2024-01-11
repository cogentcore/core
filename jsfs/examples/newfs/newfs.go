// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/hack-pad/hackpad
// Licensed under the Apache 2.0 License

//go:build js

package main

import (
	"syscall/js"

	"goki.dev/goki/grr"
	"goki.dev/goki/jsfs"
)

func main() {
	fs := grr.Must1(jsfs.NewFS())
	grr.Must1(fs.MkdirAll([]js.Value{js.ValueOf("me"), js.ValueOf(0777)}))
	js.Global().Get("console").Call("log", "stat file info", grr.Must1(fs.Stat([]js.Value{js.ValueOf("me")})))
}
