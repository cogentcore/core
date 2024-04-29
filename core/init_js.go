// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package core

import (
	"syscall/js"

	"cogentcore.org/core/events"
)

func init() {
	js.Global().Set("appOnUpdate", js.FuncOf(func(this js.Value, args []js.Value) any {
		NewBody("web-update-available").
			AddSnackbarText("A new version is available").
			AddSnackbarButton("Reload", func(e events.Event) {
				js.Global().Get("location").Call("reload")
			}).NewSnackbar(nil).SetTimeout(0).Run()
		return nil
	}))
}
