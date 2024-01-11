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
	"goki.dev/goki/grr"
	"goki.dev/goki/jsfs"
)

func main() {
	idb := grr.Must1(indexeddb.NewFS(context.Background(), "idb", indexeddb.Options{}))
	grr.Must(idb.MkdirAll("me", 0777))
	go func() {
		js.Global().Get("console").Call("log", "stat file info", jsfs.JSStat(grr.Must1(idb.Stat("me"))))
	}()
	js.Global().Get("console").Call("log", "stat file info", jsfs.JSStat(grr.Must1(idb.Stat("me"))))
	select {}
}
