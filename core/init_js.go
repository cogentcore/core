// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package core

import (
	"os"
	"syscall/js"

	"cogentcore.org/core/events"
)

func init() {
	webCrashDialog = func(title, text, body string) {
		document := js.Global().Get("document")
		div := document.Call("createElement", "div")
		h1 := document.Call("createElement", "h1")
		p := document.Call("createElement", "p")
		div.Set("id", "app-crash-dialog")
		h1.Set("innerText", title)
		p.Set("innerText", text+"\n\n"+body)
		div.Call("appendChild", h1)
		div.Call("appendChild", p)
		docBody := document.Get("body")
		docBody.Set("style", "")
		docBody.Call("appendChild", div)
		os.Exit(1)
	}

	js.Global().Set("appOnUpdate", js.FuncOf(func(this js.Value, args []js.Value) any {
		NewBody("web-update-available").
			AddSnackbarText("A new version is available").
			AddSnackbarButton("Reload", func(e events.Event) {
				js.Global().Get("location").Call("reload")
			}).NewSnackbar(nil).SetTimeout(0).Run()
		return nil
	}))

	webInstall = func() {
		js.Global().Call("appShowInstallPrompt")
	}
	if !js.Global().Get("appIsAppInstallable").IsUndefined() {
		webCanInstall = js.Global().Call("appIsAppInstallable").Bool()
	}
	js.Global().Set("appOnAppInstallChange", js.FuncOf(func(this js.Value, args []js.Value) any {
		webCanInstall = js.Global().Call("appIsAppInstallable").Bool()
		return nil
	}))
}
