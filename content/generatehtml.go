// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/content/bcontent"
	"cogentcore.org/core/core"
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
	prps := []*bcontent.PreRenderPage{}
	ct.UpdateTree() // need initial update first
	for _, pg := range ct.pages {
		ct.Open(pg.URL)
		prp := &bcontent.PreRenderPage{
			Page: *pg,
			HTML: core.GenerateHTMLCore(ct),
		}
		// The first non-emphasized paragraph is used as the description
		// (<em> typically indicates a note or caption, not an introduction).
		ct.WalkDown(func(n tree.Node) bool {
			if prp.Description != "" {
				return tree.Break
			}
			if tx, ok := n.(*core.Text); ok {
				if tx.Property("tag") == "p" && !strings.HasPrefix(tx.Text, "<em>") {
					prp.Description = tx.Text
					return tree.Break
				}
			}
			return tree.Continue
		})
		prps = append(prps, prp)
	}
	return string(errors.Log1(jsonx.WriteBytes(prps)))
}
