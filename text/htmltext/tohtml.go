// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmltext

import (
	"html"
	"slices"
	"strings"

	"cogentcore.org/core/text/rich"
)

// RichToHTML returns an HTML encoded representation of the rich.Text,
// with <b>, <i>, and <u> tags for bold, italic, and underlined text,
// <a> tags for links, and the text escaped.
func RichToHTML(tx rich.Text) string {
	var b strings.Builder
	var open []string        // the tags that are open, innermost last
	closeTo := func(n int) { // closes the tags after the first n
		for len(open) > n {
			b.WriteString("</" + open[len(open)-1] + ">")
			open = open[:len(open)-1]
		}
	}
	for si := range tx.NumSpans() {
		sty, rs := tx.Span(si)
		if sty.Special == rich.Link {
			closeTo(0)
			b.WriteString(`<a href="` + html.EscapeString(sty.URL) + `">` + html.EscapeString(string(rs)) + `</a>`)
			continue
		}
		var want []string // the tags of this span, in the order they are opened
		if tag := sty.Weight.HTMLTag(); tag != "" {
			want = append(want, tag)
		}
		if sty.Slant != rich.SlantNormal {
			want = append(want, "i")
		}
		if sty.Decoration.HasFlag(rich.Underline) {
			want = append(want, "u")
		}
		// the open tags that are wanted stay open, up to the first one that
		// is not, which is closed with the ones in it
		keep := 0
		for keep < len(open) && slices.Contains(want, open[keep]) {
			keep++
		}
		closeTo(keep)
		for _, tag := range want {
			if !slices.Contains(open, tag) {
				b.WriteString("<" + tag + ">")
				open = append(open, tag)
			}
		}
		b.WriteString(html.EscapeString(string(rs)))
	}
	closeTo(0)
	return b.String()
}
