// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"log/slog"
	"os"
)

// GenerateHTML is the Function to call for the -generatehtml argument.
// it returns generated HTML for the given widget.
// It exits the program if there is an error, but does not exit
// the program if there is no error.
var GenerateHTML func(w Widget) string

func init() {
	GenerateHTML = GenerateHTMLCore
}

func GenerateHTMLCore(w Widget) string {
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
