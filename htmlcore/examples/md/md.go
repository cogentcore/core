// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
	_ "cogentcore.org/core/text/tex" // include this to get math
)

//go:embed example.md
var content string

func main() {
	b := core.NewBody("MD Example")
	ctx := htmlcore.NewContext()
	ctx.AddWidgetHandler(func(w core.Widget) {
		switch x := w.(type) {
		case *core.Text:
			x.Styler(func(s *styles.Style) {
				s.Max.X.Ch(80)
			})
		}
	})
	errors.Log(htmlcore.ReadMDString(ctx, b, content))
	b.RunMainWindow()
}
