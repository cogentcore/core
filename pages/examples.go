// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pages

import (
	"log/slog"
	"strconv"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlview"
	"cogentcore.org/core/styles"
)

// Examples are the different core examples that exist as compiled
// Go code that can be run in pages. The map is keyed
// by ID. Generated pagegen.go files add to this by finding
// all code blocks with language Go (must be uppercase, as that
// indicates that is an "exported" example).
var Examples = map[string]func(parent core.Widget){}

func init() {
	htmlview.ElementHandlers["pages-example"] = ExampleHandler
}

// NumExamples has the number of examples per page URL.
var NumExamples = map[string]int{}

// ExampleHandler is the htmlview handler for <pages-example> HTML elements
// that handles examples.
func ExampleHandler(ctx *htmlview.Context) bool {
	// the node we actually care about is our first child, the <pre> element
	ctx.Node = ctx.Node.FirstChild

	ExampleCodeLabel(ctx.Parent()).SetText(htmlview.ExtractText(ctx))

	id := ctx.PageURL + "-" + strconv.Itoa(NumExamples[ctx.PageURL])
	NumExamples[ctx.PageURL]++
	fn := Examples[id]
	if fn == nil {
		slog.Error("programmer error: pages example not found in pages.Examples (you probably need to run go generate again)", "id", id)
	} else {
		fn(ctx.Parent())
	}

	return true
}

// ExampleCodeLabel adds a new label styled for displaying example code.
func ExampleCodeLabel(parent core.Widget) *core.Text {
	label := core.NewText(parent)
	label.Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		s.Background = colors.C(colors.Scheme.SurfaceContainer)
		s.Border.Radius = styles.BorderRadiusMedium
	})
	return label
}
