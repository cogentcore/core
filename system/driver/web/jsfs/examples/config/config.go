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

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/system/driver/web/jsfs"
	"github.com/hack-pad/hackpadfs/indexeddb"
)

func main() {
	fs := errors.Must1(jsfs.Config(js.Global().Get("fs")))
	errors.Must1(fs.MkdirAll([]js.Value{js.ValueOf("me"), js.ValueOf(0777)}))
	ifs := errors.Must1(indexeddb.NewFS(context.Background(), "/me", indexeddb.Options{}))
	errors.Must(fs.FS.AddMount("me", ifs))
	js.Global().Get("console").Call("log", "stat file info", errors.Must1(fs.Stat([]js.Value{js.ValueOf("me")})))
}
