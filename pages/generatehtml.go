// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build generatehtml

package pages

import (
	"fmt"
	"log/slog"
	"os"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
)

// This file is activated by the core tool to pre-render Cogent Core apps
// as HTML that can be used as a preview and for SEO purposes.

func init() {
	// We override the OnChildAdded set in core/generatehtml.go
	core.ExternalParent.AsWidget().SetOnChildAdded(func(n tree.Node) {
		var page *Page
		n.AsTree().WalkDown(func(n tree.Node) bool {
			if page != nil {
				return tree.Break
			}
			if pg, ok := n.(*Page); ok {
				page = pg
				return tree.Break
			}
			return tree.Continue
		})
		if page == nil {
			slog.Error("generatehtml: no pages.Page widget found in an app that imports pages")
			os.Exit(1)
		}
		fmt.Println(page)
	})
}
