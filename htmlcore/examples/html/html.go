// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
)

//go:embed example.html
var content string

func main() {
	b := core.NewBody("HTML Example")
	errors.Log(htmlcore.ReadHTMLString(htmlcore.NewContext(), b, content))
	b.RunMainWindow()
}
