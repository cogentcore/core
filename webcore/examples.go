// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webcore

import (
	"log/slog"
	"strconv"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/coredom"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/styles"
)

// Examples are the different <core-example> elements that exist,
// as compiled Go code that can be run in webcore. The map is keyed
// by ID. Generated webcoregen.go files add to this.
var Examples = map[string]func(parent gi.Widget){}

func init() {
	coredom.ElementHandlers["pre"] = ExamplePreHandler
}

// NumExamples has the number of examples per page URL.
var NumExamples = map[string]int{}

// ExamplePreHandler is the coredom handler for <pre> HTML elements
// that handles examples. It falls back on [coredom.HandleElement]
// for <pre> elements that do not have a <code> with class="language-Go".
func ExamplePreHandler(ctx *coredom.Context) bool {
	if ctx.Node.FirstChild == nil || ctx.Node.FirstChild.Data != "code" {
		return false
	}
	class := coredom.GetAttr(ctx.Node.FirstChild, "class")
	if !strings.Contains(class, "language-Go") {
		return false
	}
	fr := coredom.New[*gi.Frame](ctx)
	fr.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	gi.NewLabel(fr).SetText(coredom.ExtractText(ctx)).Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		s.Background = colors.C(colors.Scheme.SurfaceContainer)
		s.Border.Radius = styles.BorderRadiusMedium
	})

	id := ctx.PageURL + "-" + strconv.Itoa(NumExamples[ctx.PageURL])
	NumExamples[ctx.PageURL]++
	fn := Examples[id]
	if fn == nil {
		slog.Error("programmer error: core example not found in webcore.Examples (you probably need to run go generate again)", "id", id)
	} else {
		fn(fr)
	}

	return true
}
