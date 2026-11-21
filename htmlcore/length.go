// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"cogentcore.org/core/htmlcore/css"
)

// readEnv returns the length environment for parsing lengths when a page
// is read (outside of styling): an em of 16px (as the font size of the
// element is not known yet), the rem of the page, and its viewport.
func (c *Context) readEnv() *css.LengthEnv {
	e := css.DefaultLengthEnv()
	e.Rem = float64(c.rootFontSize())
	e.Vw = float64(c.mediaWidth()) / 100
	e.FixedVh = c.MediaHeight > 0
	if e.FixedVh {
		e.Vh = float64(c.MediaHeight) / 100
	}
	e.VhUsed = &c.viewportHeightUsed
	return e
}
