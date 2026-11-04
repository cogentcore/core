// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/text"
	"golang.org/x/net/html"
)

// handleBrowserElement handles the elements that are handled differently
// for [Context.BrowserStyle], returning whether it handled the element.
func handleBrowserElement(ctx *Context, tag string) bool {
	switch tag {
	case "style", "link", "script", "title", "meta", "head", "html":
	default:
		if ctx.displayOf(ctx.Node) == "none" {
			return true // not rendered, and neither are its children
		}
	}
	switch tag {
	case "html", "head":
		ctx.NewParent = ctx.BlockParent
		return true
	case "pre", "p", "h1", "h2", "h3", "h4", "h5", "h6", "hr", "dt", "dd", "figcaption", "caption", "summary", "legend":
		w := New[core.Frame](ctx)
		ctx.NewParent = w
		ctx.handleWidget(ctx.Node, w)
		if p := ctx.Node.Parent; tag == "caption" && p != nil && p.Data == "table" {
			// the caption is above the cells, across all of the columns
			span := tableColumns(p)
			w.FinalStyler(func(s *styles.Style) { s.ColSpan = span })
		}
		return true
	case "li":
		handleBrowserListItem(ctx)
		return true
	case "button":
		// a button is a block of its content, styled by CSS, that is clickable
		w := New[core.Frame](ctx)
		ctx.NewParent = w
		ctx.handleWidget(ctx.Node, w)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.Clickable)
			s.Cursor = cursors.Pointer
		})
		return true
	case "input":
		typ := strings.ToLower(GetAttr(ctx.Node, "type"))
		if typ != "submit" && typ != "button" && typ != "reset" {
			return false
		}
		label := GetAttr(ctx.Node, "value")
		if !HasAttr(ctx.Node, "value") {
			label = map[string]string{"submit": "Submit", "reset": "Reset"}[typ]
		}
		w := New[core.Frame](ctx)
		ctx.handleWidget(ctx.Node, w)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.Clickable)
			s.Cursor = cursors.Pointer
		})
		tx := core.NewText(w).SetText(html.EscapeString(label))
		ctx.handleWidget(nil, tx)
		return true
	case "svg":
		if ctx.displayOf(ctx.Node) == "none" {
			return true
		}
		var b strings.Builder
		if err := html.Render(&b, ctx.Node); err != nil {
			return true
		}
		sv := New[core.SVG](ctx)
		errors.Log(sv.ReadString(b.String()))
		ctx.handleWidget(ctx.Node, sv)
		return true
	}
	if slices.Contains(textTags, tag) { // not in a run of inline content
		switch display := ctx.displayOf(ctx.Node); display {
		case "none":
			return true
		case "", "inline":
			if !ctx.isTextElement(ctx.Node) {
				// an inline element with widgets in it, which are in the run
				// of inline content that it is in
				ctx.NewParent = ctx.Parent()
				return true
			}
			ctx.handleWidget(ctx.Node, handleTextTag(ctx))
			return true
		}
		w := New[core.Frame](ctx)
		ctx.NewParent = w
		ctx.handleWidget(ctx.Node, w)
		if href := GetAttr(ctx.Node, "href"); tag == "a" && href != "" {
			ctx.linkFrame(w, href)
		}
		return true
	}
	return false
}

// linkFrame makes the given frame of an a element open the given URL.
func (c *Context) linkFrame(w *core.Frame, href string) {
	w.SetProperty("href", href)
	w.SetTooltip(href)
	w.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable)
		s.Cursor = cursors.Pointer
	})
	w.OnClick(func(e events.Event) {
		e.SetHandled()
		c.OpenURL(href, e)
	})
}

// markerWidth is the width of the box of a list marker,
// which is in the padding of the list.
var markerWidth = units.Em(1.5)

// handleBrowserListItem handles a li element for [Context.BrowserStyle]:
// it is a row of the marker and a block of the content of the item.
// The marker is to the left of the box of the item, in the padding
// of the list, as in browsers.
func handleBrowserListItem(ctx *Context) {
	n := ctx.Node
	li := New[core.Frame](ctx)
	ctx.handleWidget(n, li)
	li.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Align.Items = styles.Start
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
	ctx.Node = n
	ctx.NewParent = content
}

// listMarker returns the marker text for the given li element.
func listMarker(ctx *Context, n *html.Node) string {
	list := n.Parent
	styleType := ""
	for p := n; p != nil && styleType == ""; p = p.Parent {
		for _, prop := range []string{"list-style-type", "list-style"} {
			if v := ctx.declaration(p, prop); v != "" {
				styleType = strings.Fields(v)[0]
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
