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
		fmt.Println(GenerateHTML(n.(Widget)))
		os.Exit(0)
	})
}

// GenerateHTML returns generated HTML for the given widget.
// It exits the program if there is an error, but does not exit
// the program if there is no error.
func GenerateHTML(w Widget) string {
	wb := w.AsWidget()
	wb.UpdateTree()
	wb.StyleTree()
	h, err := ToHTML(w)
	if err != nil {
		slog.Error("error generating pre-render HTML with generatehtml build tag", "err", err)
		os.Exit(1)
	}
	return string(h)
}
