// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package content

import (
	"fmt"
	"syscall/js"
)

// saveWebURL saves the current page URL to the user's address bar and history.
func (ct *Content) saveWebURL() {
	url := ct.currentPage.URL
	if url == "" {
		url = ".."
	}
	js.Global().Get("history").Call("pushState", "", "", url)
}

// handleWebPopState adds a JS event listener to handle user navigation in the browser.
func (ct *Content) handleWebPopState() {
	js.Global().Get("window").Call("addEventListener", "popstate", js.FuncOf(func(this js.Value, args []js.Value) any {
		url := js.Global().Get("location").Get("href").String()
		fmt.Println(url)
		return nil
	}))
}
