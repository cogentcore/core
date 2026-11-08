// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strconv"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/text"
	"golang.org/x/net/html"
)

// markerWidth is the width of the box of a list marker,
// which is in the padding of the list.
var markerWidth = units.Em(1.5)

// handleListItem handles a li element:
// it is a row of the marker and a block of the content of the item.
// The marker is to the left of the box of the item, in the padding
// of the list, as in browsers.
func handleListItem(ctx *Context) {
	n := ctx.Node
	li := New[core.Frame](ctx)
	ctx.handleWidget(n, li)
	li.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		// the marker is on the baseline of the first line of the content
		s.Align.Items = styles.Baseline
	})
	if marker := listMarker(ctx, n); marker != "" {
		mk := core.NewText(li).SetText(marker)
		mk.SetName("marker")
		ctx.handleWidget(nil, mk)
		mk.Styler(func(s *styles.Style) {
			s.Min.X = markerWidth
			s.Max.X = markerWidth
			s.Grow.X = 0
			s.GrowWrap = false
			s.Text.Align = text.End
			s.Text.WhiteSpace = text.WrapNever
		})
	}
	content := core.NewFrame(li)
	content.SetName("content")
	ctx.handleWidget(nil, content)
	if fid := li.Property("data-fid"); fid != nil { // the box of the item is that of its content
		content.SetProperty("data-fid", fid)
		delete(li.Properties, "data-fid")
	}
	ctx.blocks[content].display = "block"
	content.FinalStyler(func(s *styles.Style) { s.Shrink.X = 1 }) // beside its marker
	ctx.Node = n
	ctx.NewParent = content
}

// listMarker returns the marker text for the given li element.
func listMarker(ctx *Context, n *html.Node) string {
	list := n.Parent
	styleType := ""
	for p := n; p != nil && styleType == ""; p = p.Parent {
		// the last of list-style-type and list-style applies
		for _, d := range ctx.declarations(p) {
			switch d.Property {
			case "list-style-type":
				styleType = strings.TrimSpace(d.Value)
			case "list-style":
				if fs := strings.Fields(d.Value); len(fs) > 0 {
					styleType = fs[0]
					for _, f := range fs {
						if f != "inside" && f != "outside" && !strings.HasPrefix(f, "url(") {
							styleType = f
							break
						}
					}
				}
			}
		}
		if p == list {
			break
		}
	}
	if list == nil {
		return "• "
	}
	if styleType == "" {
		if list.Data == "ol" {
			styleType = "decimal"
		} else {
			depth := 0
			for p := list; p != nil; p = p.Parent {
				if p.Data == "ul" || p.Data == "ol" {
					depth++
				}
			}
			switch depth {
			case 1:
				styleType = "disc"
			case 2:
				styleType = "circle"
			default:
				styleType = "square"
			}
		}
	}
	switch styleType {
	case "none":
		return ""
	case "disc":
		return "• "
	case "circle":
		return "◦ "
	case "square":
		return "▪ "
	}
	// numbered
	num := 1
	if st, err := strconv.Atoi(GetAttr(list, "start")); err == nil {
		num = st
	}
	for p := list.FirstChild; p != nil && p != n; p = p.NextSibling {
		if p.Type == html.ElementNode && p.Data == "li" {
			num++
		}
	}
	if v, err := strconv.Atoi(GetAttr(n, "value")); err == nil {
		num = v
	}
	switch styleType {
	case "lower-alpha", "lower-latin":
		return string(rune('a'+(num-1)%26)) + ". "
	case "upper-alpha", "upper-latin":
		return string(rune('A'+(num-1)%26)) + ". "
	}
	return strconv.Itoa(num) + ". "
}
