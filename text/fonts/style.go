// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fonts

import (
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"github.com/go-text/typesetting/font"
)

// Style sets the [rich.Style] and [text.Style] for the given [font.Face].
func Style(face *font.Face, sty *rich.Style, tsty *text.Style) {
	if face == nil {
		return
	}
	d := face.Describe()
	tsty.CustomFont = rich.FontName(d.Family)
	sty.Family = rich.Custom
	as := d.Aspect
	sty.Weight = rich.Weights(int(as.Weight / 100.0))
	sty.Slant = rich.Slants(as.Style - 1)
	// fi[i].Stretch = rich.Stretch() // not avail
	// fi[i].Stretch = rich.StretchNormal
}
