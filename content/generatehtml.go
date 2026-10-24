// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
)

func init() {
	core.GenerateHTML = generateHTML
}

// generateHTML is called by the core tool to pre-render Cogent Core apps
// as HTML that can be used as a preview and for SEO purposes. For apps
// with [Content], it returns the JSON [bcontent.Site] data (see [generateSite]).
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
	return generateSite(ct)
}
