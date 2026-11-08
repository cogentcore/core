// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strconv"
	"strings"

	"github.com/aymerick/douceur/css"
	"golang.org/x/net/html"
)

// pseudoKey is an element and one of its ::before and ::after
// pseudo-elements.
type pseudoKey struct {
	node *html.Node
	kind string // "before" or "after"
}

// pseudoSelector returns the selector of the elements of the given
// selector of a ::before or ::after pseudo-element (also :before and
// :after), and which of them it is, or false if it is not one.
func pseudoSelector(sel string) (base, kind string, ok bool) {
	lsel := strings.ToLower(sel)
	for _, k := range []string{"before", "after"} {
		for _, p := range []string{"::" + k, ":" + k} {
			if strings.HasSuffix(lsel, p) {
				base = strings.TrimSpace(sel[:len(sel)-len(p)])
				if base == "" || strings.HasSuffix(base, " ") || strings.HasSuffix(base, ">") {
					base += "*"
				}
				return base, k, true
			}
		}
	}
	return "", "", false
}

// addPseudoRule adds the given declarations of a rule for the given
// pseudo-element of the elements under root that the given selector
// selects.
func (c *Context) addPseudoRule(root *html.Node, sel, kind string, decls []*css.Declaration, origin cssOrigin) {
	spec := specificity(sel) + 1 // the pseudo-element
	if cs := compileSelector(sel); cs != nil {
		spec = cs.specificity + 1
	} else {
		return
	}
	if c.pseudoStyles == nil {
		c.pseudoStyles = map[pseudoKey][]*cssDecl{}
	}
	for _, n := range c.selectNodes(root, sel) {
		key := pseudoKey{n, kind}
		for _, d := range decls {
			p := strings.ToLower(strings.TrimSpace(d.Property))
			switch p {
			case "content", "display", "visibility", "opacity":
			default:
				continue // only the text of pseudo-elements is supported
			}
			c.cssOrder++
			c.pseudoStyles[key] = append(c.pseudoStyles[key], &cssDecl{Property: p, Value: strings.TrimSpace(d.Value),
				Important: d.Important, origin: origin, specificity: spec, order: c.cssOrder})
		}
	}
}

// pseudoContent returns the text of the given pseudo-element ("before" or
// "after") of the given element: that of its content property, with the
// strings (with their escapes), attr() values, and quotes of it, without
// its alternative text (after a /), or "" if it has none.
func (c *Context) pseudoContent(n *html.Node, kind string) string {
	ds := c.pseudoStyles[pseudoKey{n, kind}]
	if len(ds) == 0 {
		return ""
	}
	// the winning declaration of each property, as in the cascade
	win := map[string]*cssDecl{}
	for _, d := range ds {
		w := win[d.Property]
		if w == nil || d.rank() > w.rank() || d.rank() == w.rank() && (d.specificity > w.specificity || d.specificity == w.specificity && d.order > w.order) {
			win[d.Property] = d
		}
	}
	if d := win["display"]; d != nil && strings.EqualFold(d.Value, "none") {
		return ""
	}
	if d := win["visibility"]; d != nil && (strings.EqualFold(d.Value, "hidden") || strings.EqualFold(d.Value, "collapse")) {
		return "" // (as anchors that show on hover)
	}
	if d := win["opacity"]; d != nil && strings.TrimSpace(d.Value) == "0" {
		return ""
	}
	d := win["content"]
	if d == nil {
		return ""
	}
	return contentText(n, d.Value)
}

// contentText returns the text of the given value of the content property
// for the given element.
func contentText(n *html.Node, v string) string {
	var sb strings.Builder
	for i := 0; i < len(v); {
		switch ch := v[i]; {
		case ch == '"' || ch == '\'':
			s, end := cssString(v, i)
			sb.WriteString(s)
			i = end
		case ch == '/': // the alternative text, for speech
			return sb.String()
		case ch == ' ' || ch == '\t' || ch == '\n':
			i++
		default:
			j := i
			for j < len(v) && v[j] != ' ' && v[j] != '"' && v[j] != '\'' && v[j] != '/' {
				if v[j] == '(' {
					if k := strings.IndexByte(v[j:], ')'); k >= 0 {
						j += k
					}
				}
				j++
			}
			word := strings.ToLower(v[i:j])
			switch {
			case word == "none" || word == "normal":
				return ""
			case word == "open-quote":
				sb.WriteString("“")
			case word == "close-quote":
				sb.WriteString("”")
			case strings.HasPrefix(word, "attr(") && strings.HasSuffix(word, ")"):
				sb.WriteString(GetAttr(n, strings.TrimSpace(v[i+5:j-1])))
			}
			i = j
		}
	}
	return sb.String()
}

// cssString returns the CSS string that starts at the given index of the
// given value (at its quote), with its escapes (as \2022 and \"), and the
// index after it.
func cssString(v string, start int) (string, int) {
	q := v[start]
	var sb strings.Builder
	i := start + 1
	for i < len(v) {
		ch := v[i]
		switch {
		case ch == q:
			return sb.String(), i + 1
		case ch == '\\' && i+1 < len(v):
			j := i + 1
			for j < len(v) && j < i+7 && strings.IndexByte("0123456789abcdefABCDEF", v[j]) >= 0 {
				j++
			}
			if j > i+1 { // a hexadecimal code point, with an optional space after it
				if r, err := strconv.ParseUint(v[i+1:j], 16, 32); err == nil {
					sb.WriteRune(rune(r))
				}
				if j < len(v) && v[j] == ' ' {
					j++
				}
				i = j
				continue
			}
			sb.WriteByte(v[i+1])
			i += 2
			continue
		default:
			sb.WriteByte(ch)
		}
		i++
	}
	return sb.String(), i
}
