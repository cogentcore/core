// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webcore

import (
	"log/slog"
	"strconv"

	"cogentcore.org/core/coredom"
	"cogentcore.org/core/gi"
)

// Examples are the different <core-example> elements that exist,
// as compiled Go code that can be run in webcore. The map is keyed
// by ID. Generated webcoregen.go files add to this.
var Examples = map[string]func(parent gi.Widget){}

func init() {
	coredom.ElementHandlers["core-example"] = CoreExampleHandler
}

// NumExamples has the number of examples per page URL.
var NumExamples = map[string]int{}

// CoreExampleHandler is the coredom handler for <core-example> HTML elements.
func CoreExampleHandler(ctx *coredom.Context) {
	sp := coredom.New[*gi.Splits](ctx)
	fr := gi.NewFrame(sp)

	id := ctx.PageURL + "-" + strconv.Itoa(NumExamples[ctx.PageURL])
	NumExamples[ctx.PageURL]++
	fn := Examples[id]
	if fn == nil {
		slog.Error("programmer error: <core-example> not found in webcore.Examples (you probably need to run go generate again)", "id", id)
	} else {
		fn(fr)
	}

	ctx.NewParent = sp
}
