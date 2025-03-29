// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "cogentcore.org/core/core"

func main() {
	b := core.NewBody()
	fb := NewBrowser(b)
	fb.OpenFile("../roboto/Roboto-Regular.ttf")
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(fb.MakeToolbar)
	})
	b.RunMainWindow()
}
