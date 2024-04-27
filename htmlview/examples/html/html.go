// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"

	"cogentcore.org/core/core"
	"cogentcore.org/core/gox/errors"
	"cogentcore.org/core/htmlview"
)

//go:embed example.html
var content string

func main() {
	b := core.NewBody("HTML View")
	errors.Log(htmlview.ReadHTMLString(htmlview.NewContext(), b, content))
	b.RunMainWindow()
}
