// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/hack-pad/hackpad
// Licensed under the Apache 2.0 License

//go:build js

package main

import (
	"context"
	"syscall/js"

	"github.com/hack-pad/hackpadfs/indexeddb"
	"goki.dev/grr"
	"goki.dev/jsfs"
)

func main() {
	fs := grr.Must1(jsfs.Config(js.Global().Get("fs")))
	grr.Must1(fs.MkdirAll([]js.Value{js.ValueOf("me"), js.ValueOf(0777)}))
	ifs := grr.Must1(indexeddb.NewFS(context.Background(), "/me", indexeddb.Options{}))
	grr.Must(fs.FS.AddMount("me", ifs))
	callback := js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Get("console").Call("log", "stat file info", args[1])
		return nil
	})
	js.Global().Get("fs").Call("stat", "me", callback)
	select {}
}
