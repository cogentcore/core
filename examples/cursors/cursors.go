// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/gi"
)

// TODO: fix
func main() {
	b := gi.NewBody().SetTitle("Cogent Core Cursors")
	gi.NewLabel(b).SetText("The Cogent Core Standard Cursors").SetType(gi.LabelHeadlineSmall)
	b.RunMainWindow()
}
