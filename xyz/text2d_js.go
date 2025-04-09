// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package xyz

import (
	"io/fs"

	"cogentcore.org/core/text/fonts/roboto"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
)

func initTextShaper(sc *Scene) {
	sc.TextShaper = shapedgt.NewShaperWithFonts([]fs.FS{roboto.EmbeddedFonts})
}
