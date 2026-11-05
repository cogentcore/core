// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tohtml

import (
	"log/slog"
	"os"

	"cogentcore.org/core/core"
)

func init() {
	if core.GenerateHTML == nil {
		core.GenerateHTML = Generate
	}
}

// Generate returns the HTML for the given widget (see [Widget]) after
// updating and styling it, for the -generatehtml argument (see
// [core.GenerateHTML]). It exits the program if there is an error.
func Generate(w core.Widget) string {
	wb := w.AsWidget()
	wb.UpdateTree()
	wb.StyleTree()
	h, err := Widget(w)
	if err != nil {
		slog.Error("error generating pre-render HTML", "err", err)
		os.Exit(1)
	}
	return string(h)
}
