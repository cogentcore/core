// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build generatehtml

package core

import (
	"fmt"
	"log/slog"
	"os"

	"cogentcore.org/core/tree"
)

// This file is activated by the core tool to pre-render Cogent Core apps
// as HTML that can be used as a preview and for SEO purposes.

func init() {
	wb := NewWidgetBase()
	ExternalParent = wb
	wb.SetOnChildAdded(func(n tree.Node) {
		bd := n.(*Body)
		bd.UpdateTree()
		bd.StyleTree()
		h, err := ToHTML(bd)
		if err != nil {
			slog.Error("error generating pre-render HTML with generatehtml build tag", "err", err)
			os.Exit(1)
		}
		fmt.Println(string(h))
		os.Exit(0)
	})
}
