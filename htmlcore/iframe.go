// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// IframeDefaultWidth and IframeDefaultHeight are the size in CSS pixels
// of an iframe without a width or height, as in browsers.
const (
	IframeDefaultWidth  = 300
	IframeDefaultHeight = 150
)

// handleIframe handles an iframe element: a box of its size (300 by 150
// by default, as a replaced element), with the border of browsers unless
// it has frameborder="0". Its children are only for browsers without
// frames, and are not shown. The document in it is not read here: an app
// that shows them (as a web browser) puts it in the frame of the iframe
// from a widget handler (see [Context.WidgetHandlers]), as it has its own
// scripts and history.
func handleIframe(ctx *Context) core.Widget {
	fr := New[core.Frame](ctx)
	fb := strings.ToLower(strings.TrimSpace(GetAttr(ctx.Node, "frameborder")))
	border := fb != "0" && fb != "no"
	fr.Styler(func(s *styles.Style) {
		// (its size is that of a replaced element: see [resolver.sizes])
		s.Overflow.Set(styles.OverflowHidden) // its document scrolls itself
		s.Padding.Zero()
		s.Gap.Zero()
		s.Direction = styles.Column
		if border {
			s.Border.Width.Set(units.Dp(2))
			s.Border.Style.Set(styles.BorderInset)
			s.Border.Color.Set(colors.Uniform(colors.Gray))
		}
	})
	return fr
}
