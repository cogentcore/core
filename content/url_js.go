// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package content

import "syscall/js"

func init() {
	saveWebURL = func(ct *Content) {
		js.Global().Get("history").Call("pushState", "", "", ct.currentPage.URL)
	}
}
