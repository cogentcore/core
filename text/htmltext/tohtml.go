// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmltext

import (
	"strings"

	"cogentcore.org/core/text/rich"
)

// RichToHTML returns an HTML encoded representation of the rich.Text.
func RichToHTML(tx rich.Text) string {
	var b strings.Builder
	ns := tx.NumSpans()
	var lsty *rich.Style
	for si := range ns {
		sty, rs := tx.Span(si)
		var stags, etags string
		if sty.Weight != rich.Normal && (lsty == nil || lsty.Weight != sty.Weight) {
			stags += "<" + sty.Weight.HTMLTag() + ">"
		} else if sty.Weight == rich.Normal && (lsty != nil && lsty.Weight != sty.Weight) {
			etags += "</" + lsty.Weight.HTMLTag() + ">"
		}
		if sty.Slant != rich.SlantNormal && (lsty == nil || lsty.Slant != sty.Slant) {
			stags += "<i>"
		} else if sty.Slant == rich.SlantNormal && lsty != nil && lsty.Slant != sty.Slant {
			etags += "</i>"
		}
		if sty.Decoration.HasFlag(rich.Underline) && (lsty == nil || !lsty.Decoration.HasFlag(rich.Underline)) {
			stags += "<u>"
		} else if !sty.Decoration.HasFlag(rich.Underline) && lsty != nil && lsty.Decoration.HasFlag(rich.Underline) {
			etags += "</u>"
		}
		b.WriteString(etags)
		b.WriteString(stags)
		b.WriteString(string(rs))
		lsty = sty
	}
	return b.String()
}
