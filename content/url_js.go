// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package content

import "syscall/js"

// saveWebURL saves the current page URL to the user's address bar and history.
func (ct *Content) saveWebURL() {
	url := ct.currentPage.URL
	if url == "" {
		url = ".."
	}
	js.Global().Get("history").Call("pushState", "", "", url)
}
