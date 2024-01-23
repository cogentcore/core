// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"

	"cogentcore.org/core/coredom"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
)

//go:embed example.html
var content string

func main() {
	b := gi.NewAppBody("Coredom HTML")
	grr.Log(coredom.ReadHTMLString(coredom.NewContext(), b, content))
	b.StartMainWindow()
}
