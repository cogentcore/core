// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/content/bcontent"
	"cogentcore.org/core/core"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

func init() {
	core.GenerateHTML = generateHTML
}

// GenerateHTML is called by the core tool to pre-render Cogent Core apps
// as HTML that can be used as a preview and for SEO purposes.
func generateHTML(w core.Widget) string {
	var ct *Content
	w.AsTree().WalkDown(func(n tree.Node) bool {
		if ct != nil {
			return tree.Break
		}
		if c, ok := n.(*Content); ok {
			ct = c
			return tree.Break
		}
		return tree.Continue
	})
	if ct == nil {
		return core.GenerateHTMLCore(w) // basic fallback
	}
	if system.HTMLOnlyArg() {
		return generateHTMLOnly(ct)
	}
	prps := []*bcontent.PreRenderPage{}
	ct.UpdateTree() // need initial update first
	for _, pg := range ct.pages {
		ct.Open(pg.URL)
		prp := &bcontent.PreRenderPage{
			Page:        *pg,
			HTML:        core.GenerateHTMLCore(ct),
			Description: pageDescription(ct),
		}
		prps = append(prps, prp)
	}
	return string(errors.Log1(jsonx.WriteBytes(prps)))
}
