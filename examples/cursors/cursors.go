// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/gi"
)

// TODO: fix
func main() {
	b := gi.NewBody().SetTitle("Goki Cursors")
	gi.NewLabel(b).SetText("The Goki Standard Cursors").SetType(gi.LabelHeadlineSmall)
	b.NewWindow().Run().Wait()
}
