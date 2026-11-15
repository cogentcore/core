// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package renderers sets the standard renderers of the platform for the
// paint package: images, SVG, and PDF (importing it is enough).
package renderers

import (
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/renderers/pdfrender"
	"cogentcore.org/core/paint/renderers/svgrender"
	_ "cogentcore.org/core/text/shaped/shapers"
)

func init() {
	paint.NewSVGRenderer = svgrender.New
	paint.NewPDFRenderer = pdfrender.New
}
