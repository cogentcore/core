// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html"
)

// arrangeDetails makes the given widget for a details element open and
// close on a click of its summary (the first summary element in it), as
// browsers do without JavaScript: when it is closed (without the open
// attribute, at first), only its summary is shown. The summary has a
// disclosure marker (▸ or ▾), unless its list-style (type) is none.
func (c *Context) arrangeDetails(n *html.Node, fr *core.Frame) {
	var summary *core.Frame
	for _, k := range fr.Children {
		if kf := frameOf(k); kf != nil && kf.Property("tag") == "summary" {
			summary = kf
			break
		}
	}
	open := HasAttr(n, "open")
	for _, k := range fr.Children {
		if k == tree.Node(summary) {
			continue
		}
		if kw, ok := k.(core.Widget); ok {
			kw.AsWidget().FinalStyler(func(s *styles.Style) {
				if !open {
					s.Display = styles.DisplayNone
				}
			})
		}
	}
	if summary == nil {
		return
	}
	var marker *core.Text
	if sn, _ := summary.Property("node").(*html.Node); sn != nil && c.declaration(sn, "display") == "list-item" &&
		!c.noListMarker(sn) {
		marker = core.NewText(summary).SetText("▸ ")
		c.handleWidget(nil, marker)
		kids := summary.Children
		summary.Children = append([]tree.Node{marker}, kids[:len(kids)-1]...) // first
		summary.FinalStyler(func(s *styles.Style) {
			s.Direction = styles.Row // beside its content
			s.Align.Items = styles.Start
		})
	}
	setMarker := func() {
		if marker != nil {
			if open {
				marker.SetText("▾ ")
			} else {
				marker.SetText("▸ ")
			}
		}
	}
	setMarker()
	summary.Styler(func(s *styles.Style) {
		// hoverable for summary:hover (as the highlight of the items of
		// the sidebar of MDN)
		s.SetAbilities(true, abilities.Clickable, abilities.Hoverable)
		s.Cursor = cursors.Pointer
	})
	toggle := c.ToggleDetails
	summary.OnClick(func(e events.Event) {
		if toggle != nil && toggle(n) { // in the document, which is read again
			return
		}
		e.SetHandled()
		open = !open
		setMarker()
		fr.Update()
	})
}

// noListMarker returns whether the given list item has no marker, by a
// list-style-type of none, or a list-style shorthand with none and no image
// (as the summaries of the tooltips of pkg.go.dev, with list-style: none).
func (c *Context) noListMarker(n *html.Node) bool {
	// the longhand or the shorthand, whichever is last in the cascade
	prop, v := c.lastDeclaration(n, "list-style-type", "list-style")
	v = strings.ToLower(v)
	if prop == "list-style-type" {
		return strings.TrimSpace(v) == "none"
	}
	return slices.Contains(strings.Fields(v), "none") && !strings.Contains(v, "url(")
}
