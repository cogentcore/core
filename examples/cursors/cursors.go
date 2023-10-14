// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
)

func main() { gimain.Run(app) }

func app() {
	scene := gi.NewScene().SetTitle("Cursors")
	gi.NewLabel(scene).SetText("The GoGi Standard Cursors").SetType(gi.LabelHeadlineSmall)
	gi.NewWindow(scene).Run().Wait()
}
