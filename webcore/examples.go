// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webcore

import (
	"log/slog"
	"strconv"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/coredom"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/styles"
)

// Examples are the different core examples that exist as compiled
// Go code that can be run in webcore. The map is keyed
// by ID. Generated webcoregen.go files add to this by finding
// all code blocks with language Go (must be uppercase, as that
// indicates that is an "exported" example).
var Examples = map[string]func(parent gi.Widget){}

func init() {
	coredom.ElementHandlers["webcore-example"] = ExampleHandler
}

// NumExamples has the number of examples per page URL.
var NumExamples = map[string]int{}

// ExampleHandler is the coredom handler for <webcore-example> HTML elements
// that handles examples.
func ExampleHandler(ctx *coredom.Context) bool {
	// the node we actually care about is our first child, the <pre> element
	ctx.Node = ctx.Node.FirstChild

	gi.NewLabel(ctx.Parent()).SetText(coredom.ExtractText(ctx)).Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		s.Background = colors.C(colors.Scheme.SurfaceContainer)
		s.Border.Radius = styles.BorderRadiusMedium
	})

	id := ctx.PageURL + "-" + strconv.Itoa(NumExamples[ctx.PageURL])
	NumExamples[ctx.PageURL]++
	fn := Examples[id]
	if fn == nil {
		slog.Error("programmer error: webcore example not found in webcore.Examples (you probably need to run go generate again)", "id", id)
	} else {
		fn(ctx.Parent())
	}

	return true
}
