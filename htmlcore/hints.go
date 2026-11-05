// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// addHints adds declarations for the presentational attributes of the
// given element, such as the width of an img or the bgcolor of a td.
func (c *Context) addHints(n *html.Node) {
	for _, a := range n.Attr {
		v := strings.TrimSpace(a.Val)
		switch a.Key {
		case "width", "height":
			switch n.Data {
			case "img", "table", "td", "th", "col", "iframe", "video", "canvas", "embed", "object", "hr", "svg":
				c.addHint(n, a.Key, v)
			}
		case "bgcolor":
			c.addHint(n, "background-color", v)
		case "color":
			if n.Data == "font" {
				c.addHint(n, "color", v)
			}
		case "face":
			if n.Data == "font" {
				c.addHint(n, "font-family", v)
			}
		case "size":
			if n.Data == "font" {
				if sz, ok := fontElementSize(v); ok {
					c.addHint(n, "font-size", sz)
				}
			}
		case "valign":
			c.addHint(n, "vertical-align", strings.ToLower(v))
		case "text":
			if n.Data == "body" {
				c.addHint(n, "color", v)
			}
		case "align":
			switch strings.ToLower(v) {
			case "center", "middle":
				if n.Data == "table" {
					c.addHint(n, "margin-left", "auto")
					c.addHint(n, "margin-right", "auto")
				} else if n.Data != "img" {
					c.addHint(n, "text-align", "-webkit-center")
				}
			case "left", "right", "justify":
				if n.Data != "img" && n.Data != "table" {
					c.addHint(n, "text-align", strings.ToLower(v))
				}
			}
		case "hidden":
			c.addHint(n, "display", "none")
		case "nowrap":
			c.addHint(n, "white-space", "nowrap")
		case "border":
			if n.Data == "table" {
				if bw, err := strconv.Atoi(v); err == nil || v == "" {
					if v == "" {
						bw = 1
					}
					if bw > 0 {
						c.addHint(n, "border", strconv.Itoa(bw)+"px outset #808080")
					}
				}
			} else if n.Data == "img" {
				c.addHint(n, "border-width", v)
			}
		case "cellspacing":
			if n.Data == "table" {
				c.addHint(n, "border-spacing", v)
			}
		}
	}
	// cells get the cellpadding and border of their table
	if n.Data == "td" || n.Data == "th" {
		if t := ancestorTable(n); t != nil {
			if cp := GetAttr(t, "cellpadding"); cp != "" {
				c.addHint(n, "padding", cp)
			}
			if b := GetAttr(t, "border"); HasAttr(t, "border") && b != "0" {
				c.addHint(n, "border", "1px inset #808080")
			}
		}
	}
}

// fontElementSize returns the CSS font size for the size attribute of a font element.
func fontElementSize(v string) (string, bool) {
	sizes := []string{"x-small", "small", "medium", "large", "x-large", "xx-large", "xxx-large"}
	n, err := strconv.Atoi(strings.TrimLeft(v, "+-"))
	if err != nil {
		return "", false
	}
	switch {
	case strings.HasPrefix(v, "+"):
		n += 3
	case strings.HasPrefix(v, "-"):
		n = 3 - n
	}
	return sizes[min(max(n, 1), 7)-1], true
}

// ancestorTable returns the table element that the given cell is in.
func ancestorTable(n *html.Node) *html.Node {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Data == "table" {
			return p
		}
	}
	return nil
}
