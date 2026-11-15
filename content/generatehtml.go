// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore/tohtml"
)

func init() {
	core.GenerateHTML = generateHTML
}

// generateHTML is called by the core tool to pre-render Cogent Core apps
// as HTML that can be used as a preview and for SEO purposes. For apps
// with [Content], it returns the JSON [bcontent.Site] data (see [generateSite]).
func generateHTML(w core.Widget) string {
	var ct *Content
	for n := range w.AsTree().All() {
		if c, ok := n.(*Content); ok {
			ct = c
			break
		}
	}
	if ct == nil {
		return tohtml.Generate(w) // basic fallback
	}
	return generateSite(ct)
}
