// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build generatehtml

package content

import (
	"fmt"
	"os"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/pages/ppath"
	"cogentcore.org/core/tree"
)

// This file is activated by the core tool to pre-render Cogent Core apps
// as HTML that can be used as a preview and for SEO purposes.

func init() {
	// We override the OnChildAdded set in core/generatehtml.go
	core.ExternalParent.AsWidget().SetOnChildAdded(func(n tree.Node) {
		var ct *Content
		n.AsTree().WalkDown(func(n tree.Node) bool {
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
			fmt.Println(core.GenerateHTML(n.(core.Widget))) // basic fallback
			os.Exit(0)
		}
		data := &ppath.PreRenderData{
			Description: map[string]string{},
			HTML:        map[string]string{},
		}
		ct.UpdateTree() // need initial update first
		for _, pg := range ct.pages {
			ct.Open(pg.URL)
			data.HTML[pg.URL] = core.GenerateHTML(ct)
			desc := ""
			// The first non-emphasized paragraph is used as the description
			// (<em> typically indicates a note or caption, not an introduction).
			ct.WalkDown(func(n tree.Node) bool {
				if desc != "" {
					return tree.Break
				}
				if tx, ok := n.(*core.Text); ok {
					if tx.Property("tag") == "p" && !strings.HasPrefix(tx.Text, "<em>") {
						desc = tx.Text
						return tree.Break
					}
				}
				return tree.Continue
			})
			data.Description[pg.URL] = desc
		}
		fmt.Println(string(errors.Log1(jsonx.WriteBytes(data))))
	})
}
