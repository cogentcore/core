// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/hack-pad/hackpad
// Licensed under the Apache 2.0 License

//go:build js

package main

import (
	"context"
	"syscall/js"

	"cogentcore.org/core/errors"
	"cogentcore.org/core/jsfs"
	"github.com/hack-pad/hackpadfs/indexeddb"
)

func main() {
	idb := errors.Must1(indexeddb.NewFS(context.Background(), "idb", indexeddb.Options{}))
	errors.Must(idb.MkdirAll("me", 0777))
	js.Global().Get("console").Call("log", "stat file info", jsfs.JSStat(errors.Must1(idb.Stat("me"))))
}
