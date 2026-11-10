// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strconv"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
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
	c.pseudoRules = append(c.pseudoRules, &pseudoRule{root: root, sel: compileSelector(sel), kind: kind, decls: decls, origin: origin, spec: spec})
	if strings.Contains(sel, ":has(") {
		c.relational = true
	}
	if c.recordOnly {
		return // (see Context.addedStyle)
	}
	for _, n := range c.selectNodes(root, sel) {
		key := pseudoKey{n, kind}
		for _, d := range decls {
			p := strings.ToLower(strings.TrimSpace(d.Property))
			c.cssOrder++
			c.pseudoStyles[key] = append(c.pseudoStyles[key], &cssDecl{Property: p, Value: c.absoluteURLs(strings.TrimSpace(d.Value)),
				Important: d.Important, origin: origin, specificity: spec, order: c.cssOrder})
		}
	}
}

// pseudoBox returns whether the given pseudo-element ("before" or "after")
// of the given element is a box, which has a widget of its own (see
// [Context.pseudoBoxWidget]): it has content without text (as "", for the
// decorations of pages, as the line under the title of Wikipedia), is
// displayed, and has a background, a border, a size, padding, or an
// absolute or fixed position.
func (c *Context) pseudoBox(n *html.Node, kind string) bool {
	win := c.pseudoDecls(n, kind)
	if win == nil {
		return false
	}
	ct := win["content"]
	if ct == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(ct.Value)) {
	case "none", "normal":
		return false
	}
	if strings.TrimSpace(contentText(n, ct.Value)) != "" {
		return false // text, in the text of the element (see [Context.pseudoMarkup])
	}
	if d := win["display"]; d != nil && strings.EqualFold(strings.TrimSpace(d.Value), "none") {
		return false
	}
	if d := win["position"]; d != nil {
		switch strings.ToLower(strings.TrimSpace(d.Value)) {
		case "absolute", "fixed":
			return true
		}
	}
	for prop, d := range win {
		v := strings.ToLower(strings.TrimSpace(d.Value))
		switch {
		case prop == "background" || prop == "background-color" || prop == "background-image":
			if v != "none" && v != "transparent" && v != "" {
				return true
			}
		case strings.HasPrefix(prop, "border") && !strings.Contains(prop, "radius"):
			if v != "none" && v != "0" && v != "" {
				return true
			}
		case prop == "width" || prop == "height" || prop == "min-width" || prop == "min-height" || prop == "aspect-ratio":
			if v != "auto" && v != "" && v != "0" {
				return true
			}
		case strings.HasPrefix(prop, "padding"):
			// as the boxes of the aspect ratios of the images of
			// edition.cnn.com, of padding-bottom: 56.25%
			if strings.Trim(v, "0 .pxem%") != "" || strings.ContainsAny(v, "123456789") {
				return true
			}
		}
	}
	return false
}

// pseudoAttr is the attribute of the elements of the boxes of
// pseudo-elements (see [Context.pseudoBoxNode]), which no element of a
// document has, and which no selector matches.
const pseudoAttr = ":pseudo"

// PseudoElement returns the kind of the pseudo-element ("before" or
// "after") whose box the given element is (an element that is not in the
// document; see the widgets of [NodeOf]), or "" if it is not one, as for
// an app that gives the elements of the document events: the clicks on a
// pseudo-element are those of its element (as the chevrons of the sidebar
// of MDN).
func PseudoElement(n *html.Node) string {
	if n == nil || n.Type != html.ElementNode || len(n.Attr) != 1 || n.Attr[0].Key != pseudoAttr {
		return ""
	}
	return n.Attr[0].Val
}

// pseudoBoxNode returns an element for the given pseudo-element of the
// given element if it is a box (see [Context.pseudoBox]), or nil: an
// element that is not in the document, with the declarations of the
// pseudo-element, which is read as a child of the element.
func (c *Context) pseudoBoxNode(el *html.Node, kind string) *html.Node {
	if !c.pseudoBox(el, kind) {
		return nil
	}
	pn := &html.Node{Type: html.ElementNode, Data: "span", Attr: []html.Attribute{{Key: pseudoAttr, Val: kind}}}
	var decls []*cssDecl
	for _, d := range c.pseudoStyles[pseudoKey{el, kind}] {
		if d.Property != "content" {
			nd := *d
			decls = append(decls, &nd)
		}
	}
	c.styles[pn] = decls
	c.varMemo = nil
	if c.hinted == nil {
		c.hinted = map[*html.Node]bool{}
	}
	c.hinted[pn] = true // no attributes
	return pn
}

// pseudoDecls returns the winning declaration of each property of the
// given pseudo-element ("before" or "after") of the given element, as in
// the cascade, or nil if it has none.
func (c *Context) pseudoDecls(n *html.Node, kind string) map[string]*cssDecl {
	ds := c.pseudoStyles[pseudoKey{n, kind}]
	if len(ds) == 0 {
		return nil
	}
	win := map[string]*cssDecl{}
	for _, d := range ds {
		w := win[d.Property]
		if w == nil || d.rank() > w.rank() || d.rank() == w.rank() && (d.specificity > w.specificity || d.specificity == w.specificity && d.order > w.order) {
			win[d.Property] = d
		}
	}
	return win
}

// pseudoMarkup returns the rich text markup of the given pseudo-element
// ("before" or "after") of the given element: its text, in its color, with
// its horizontal margins and padding as spaces of their widths (as the
// space between the brackets and the edit links of Wikipedia), or "".
func (c *Context) pseudoMarkup(n *html.Node, kind string) string {
	t := c.pseudoContent(n, kind)
	if t == "" || c.pseudoBox(n, kind) { // (a box has a widget of its own)
		return ""
	}
	win := c.pseudoDecls(n, kind)
	// the margin and padding of each side: the declaration of it that is
	// last in the cascade (as margin-inline after margin-right)
	type side struct {
		v     string
		order int
	}
	sides := map[string]side{} // margin-left, padding-right, ...
	set := func(d *cssDecl, prop, v string) {
		if s, ok := sides[prop]; !ok || d.order > s.order {
			sides[prop] = side{v, d.order}
		}
	}
	for _, pre := range []string{"margin", "padding"} {
		if d := win[pre]; d != nil {
			fs := strings.Fields(d.Value)
			switch len(fs) {
			case 1:
				set(d, pre+"-left", fs[0])
				set(d, pre+"-right", fs[0])
			case 2, 3:
				set(d, pre+"-left", fs[1])
				set(d, pre+"-right", fs[1])
			case 4:
				set(d, pre+"-left", fs[3])
				set(d, pre+"-right", fs[1])
			}
		}
		if d := win[pre+"-inline"]; d != nil {
			fs := strings.Fields(d.Value)
			if len(fs) == 1 {
				fs = append(fs, fs[0])
			}
			set(d, pre+"-left", fs[0])
			set(d, pre+"-right", fs[1])
		}
		for _, sd := range [][2]string{{"-left", "-left"}, {"-inline-start", "-left"}, {"-right", "-right"}, {"-inline-end", "-right"}} {
			if d := win[pre+sd[0]]; d != nil {
				set(d, pre+sd[1], d.Value)
			}
		}
	}
	left := emLength(sides["margin-left"].v) + emLength(sides["padding-left"].v)
	right := emLength(sides["margin-right"].v) + emLength(sides["padding-right"].v)
	m := html.EscapeString(t)
	// the word joiners keep the spaces from being trimmed (at the start
	// or end of a text)
	if sp := emSpaces(left); sp != "" {
		m = "\u2060" + sp + m
	}
	if sp := emSpaces(right); sp != "" {
		m += sp + "\u2060"
	}
	if d := win["color"]; d != nil {
		clr, ok := c.substituteVars(n, d.Value, nil)
		if ok && clr != "" && !strings.ContainsAny(clr, `"<>`) {
			m = `<span style="color: ` + clr + `">` + m + `</span>`
		}
	}
	return m
}

// emLength returns the given CSS length in em, with px for 16px to the
// em (and rem as em), or 0 if it is not one of these.
func emLength(v string) float64 {
	v = strings.TrimSpace(strings.ToLower(v))
	for _, u := range []struct {
		suffix string
		scale  float64
	}{{"rem", 1}, {"em", 1}, {"px", 1.0 / 16}} {
		if num, ok := strings.CutSuffix(v, u.suffix); ok {
			f, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return 0
			}
			return f * u.scale
		}
	}
	return 0
}

// emSpaces returns a string of space characters (which are not collapsed
// or trimmed) about as wide as the given number of em.
func emSpaces(em float64) string {
	var sb strings.Builder
	for _, sp := range []struct {
		r rune
		w float64
	}{{'\u2003', 1}, {'\u2002', 0.5}, {'\u2004', 1.0 / 3}, {'\u2005', 0.25}, {'\u2006', 1.0 / 6}, {'\u200A', 0.1}} {
		for em >= sp.w-0.04 && sb.Len() < 64 {
			sb.WriteRune(sp.r)
			em -= sp.w
		}
	}
	return sb.String()
}

// pseudoContent returns the text of the given pseudo-element ("before" or
// "after") of the given element: that of its content property, with the
// strings (with their escapes), attr() values, and quotes of it, without
// its alternative text (after a /), or "" if it has none.
func (c *Context) pseudoContent(n *html.Node, kind string) string {
	win := c.pseudoDecls(n, kind)
	if win == nil {
		return ""
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

// pseudoText adds a text with the content of the ::before or ::after (the
// given kind) of the given element to the given parent, if it has any.
func (c *Context) pseudoText(parent core.Widget, el *html.Node, kind string) {
	if strings.TrimSpace(c.pseudoContent(el, kind)) == "" || c.pseudoBox(el, kind) {
		return
	}
	tx := core.NewText(parent).SetText(c.pseudoMarkup(el, kind))
	tx.SetName(kind)
	tree.SetUniqueName(tx)
	c.handleWidget(nil, tx)
}
