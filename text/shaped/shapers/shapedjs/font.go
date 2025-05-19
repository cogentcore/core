// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"fmt"
	"strings"
	"syscall/js"

	"cogentcore.org/core/text/text"
)

// SetFontStyle sets the html canvas font style from [text.Font] and
// [text.Style], with optional lineHeight
func SetFontStyle(ctx js.Value, fnt *text.Font, tsty *text.Style, lineHeight float32) {
	// See https://developer.mozilla.org/en-US/docs/Web/CSS/font
	fsty := fnt.Style(tsty)
	fsz := ""
	if lineHeight > 0 {
		fsz = fmt.Sprintf("%gpx/%gpx", fnt.Size, lineHeight)
	} else {
		fsz = fmt.Sprintf("%gpx", fnt.Size)
	}
	fam := tsty.FontFamily(fsty)
	// note: no fsty.Stretch.String(), here:
	// font: font-style font-variant font-weight font-size/line-height font-family
	parts := []string{fsty.Slant.String(), "normal", fmt.Sprintf("%g", fsty.Weight.ToFloat32()), fsz, fam}
	fspec := strings.Join(parts, " ")
	// fmt.Println("measure:", tsty.FontSize.Dots, fnt.Size, lineHeight, fam)
	// fmt.Println("fspec:", fspec)
	ctx.Set("font", fspec)
}
