// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strconv"
	"strings"

	"cogentcore.org/core/htmlcore/css"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"golang.org/x/net/html"
)

// setFontSize sets the font size, relative to the inherited one.
func setFontSize(env *css.LengthEnv, s, ps *styles.Style, v string) {
	inherited := s.Font.Size
	if ps != nil {
		inherited = ps.Font.Size
		inherited.Value /= core.AppearanceSettings.FontSize / 100
	}
	scale := func(f float32) {
		s.Font.Size = units.Value{Value: inherited.Value * f, Unit: inherited.Unit}
	}
	if px, ok := css.FontSizes[v]; ok {
		s.Font.Size = units.Dp(px)
		return
	}
	switch v {
	case "smaller":
		scale(1 / 1.2)
		return
	case "larger":
		scale(1.2)
		return
	}
	if before, ok := strings.CutSuffix(v, "%"); ok {
		if f, err := strconv.ParseFloat(before, 32); err == nil {
			scale(float32(f) / 100)
		}
		return
	}
	if before, ok := strings.CutSuffix(v, "rem"); ok {
		if f, err := strconv.ParseFloat(before, 32); err == nil {
			s.Font.Size = units.Dp(float32(env.Rem) * float32(f))
		}
		return
	}
	if before, ok := strings.CutSuffix(v, "em"); ok {
		if f, err := strconv.ParseFloat(before, 32); err == nil {
			scale(float32(f))
		}
		return
	}
	if l, ok := env.Length(v, units.UnitDp); ok {
		s.Font.Size = l
	}
}

// setFont sets the font shorthand: [style] [weight] size[/line-height] family,
// returning the family list if it is valid.
func setFont(st *cssState, s, ps *styles.Style, v string) string {
	// the shorthand resets the properties that it does not set, as
	// font-weight to normal, once it is valid (with a size and family)
	fs := strings.Fields(v)
	slant, weight := rich.SlantNormal, ""
	for i, f := range fs {
		lf := strings.ToLower(f)
		switch {
		case lf == "italic" || lf == "oblique":
			slant = rich.Italic
		case lf == "bold" || lf == "bolder" || lf == "lighter" || (len(lf) == 3 && lf[1] == '0' && lf[2] == '0'):
			weight = lf
		case lf == "normal" || lf == "small-caps":
		case len(lf) > 0 && (lf[0] >= '0' && lf[0] <= '9' || lf[0] == '.') || css.FontSizes[lf] > 0:
			s.Font.Slant = slant
			s.Font.Weight = rich.Normal
			if weight != "" {
				css.SetFontWeight(s, ps, weight)
			}
			size, lh, _ := strings.Cut(lf, "/")
			setFontSize(st.env, s, ps, size)
			st.lineHeight = "normal"
			if lh != "" {
				st.lineHeight = lh
			}
			family := strings.Join(fs[i+1:], " ")
			s.Font.Family = css.FontFamily(family)
			return family
		}
	}
	return ""
}

// rootFontSize returns the font size of the html element of the page in px,
// which rem lengths are relative to (16 unless the page sets it), as the
// html element has no widget.
func (c *Context) rootFontSize() float32 {
	if c.rootFont > 0 {
		return c.rootFont
	}
	c.rootFont = 16
	if c.Node == nil {
		return c.rootFont
	}
	root := rootNode(c.Node)
	var htmlNode *html.Node
	if root.Type == html.ElementNode && root.Data == "html" {
		htmlNode = root
	}
	for n := root.FirstChild; n != nil && htmlNode == nil; n = n.NextSibling {
		if n.Type == html.ElementNode && n.Data == "html" {
			htmlNode = n
		}
	}
	if htmlNode == nil {
		return c.rootFont
	}
	v := strings.TrimSpace(strings.ToLower(c.declaration(htmlNode, "font-size")))
	num := func(suffix string) (float32, bool) {
		f, err := strconv.ParseFloat(strings.TrimSuffix(v, suffix), 32)
		return float32(f), err == nil && strings.HasSuffix(v, suffix)
	}
	if f, ok := num("%"); ok {
		c.rootFont = 16 * f / 100
	} else if f, ok := num("rem"); ok {
		c.rootFont = 16 * f
	} else if f, ok := num("em"); ok {
		c.rootFont = 16 * f
	} else if f, ok := num("px"); ok {
		c.rootFont = f
	} else if px, ok := css.FontSizeKeywords[v]; ok {
		c.rootFont = px
	}
	return c.rootFont
}
