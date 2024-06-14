// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pages

import (
	"log/slog"
	"strconv"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
)

// Examples are the different core examples that exist as compiled
// Go code that can be run in pages. The map is keyed
// by ID. Generated pagegen.go files add to this by finding
// all code blocks with language Go (must be uppercase, as that
// indicates that is an "exported" example).
var Examples = map[string]func(parent core.Widget){}

func init() {
	htmlcore.ElementHandlers["pages-example"] = ExampleHandler
}

// NumExamples has the number of examples per page URL.
var NumExamples = map[string]int{}

// ExampleHandler is the htmlcore handler for <pages-example> HTML elements
// that handles examples.
func ExampleHandler(ctx *htmlcore.Context) bool {
	// the node we actually care about is our first child, the <pre> element
	ctx.Node = ctx.Node.FirstChild

	core.NewText(ctx.Parent()).SetText(htmlcore.ExtractText(ctx)).Styler(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		s.Background = colors.C(colors.Scheme.SurfaceContainer)
		s.Border.Radius = styles.BorderRadiusMedium
	})

	url := ctx.PageURL
	if url == "" {
		url = "index"
	}
	id := url + "-" + strconv.Itoa(NumExamples[ctx.PageURL])
	NumExamples[ctx.PageURL]++
	fn := Examples[id]
	if fn == nil {
		slog.Error("programmer error: pages example not found in pages.Examples (you probably need to run go generate again)", "id", id)
	} else {
		fn(ctx.Parent())
	}

	return true
}
