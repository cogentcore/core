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
	"cogentcore.org/core/text/rich"
	"golang.org/x/net/html"
)

// lineHeightNormal is the line height of the "normal" value, which
// is given by the metrics of the font in browsers; this is the value
// for the default Noto Sans font.
const lineHeightNormal = 1.362

// cssFontSizes are the absolute font size keywords, in px.
var cssFontSizes = map[string]float32{
	"xx-small": 9, "x-small": 10, "small": 13, "medium": 16, "large": 18,
	"x-large": 24, "xx-large": 32, "xxx-large": 48,
}

// setFontSize sets the font size, relative to the inherited one.
func setFontSize(env *lengthEnv, s, ps *styles.Style, v string) {
	inherited := s.Font.Size
	if ps != nil {
		inherited = ps.Font.Size
		inherited.Value /= core.AppearanceSettings.FontSize / 100
	}
	scale := func(f float32) {
		s.Font.Size = units.Value{Value: inherited.Value * f, Unit: inherited.Unit}
	}
	if px, ok := cssFontSizes[v]; ok {
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
	if strings.HasSuffix(v, "%") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(v, "%"), 32); err == nil {
			scale(float32(f) / 100)
		}
		return
	}
	if strings.HasSuffix(v, "rem") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(v, "rem"), 32); err == nil {
			s.Font.Size = units.Dp(float32(env.rem) * float32(f))
		}
		return
	}
	if strings.HasSuffix(v, "em") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(v, "em"), 32); err == nil {
			scale(float32(f))
		}
		return
	}
	if l, ok := env.length(v, units.UnitDp); ok {
		s.Font.Size = l
	}
}

// setFontWeight sets the font weight.
func setFontWeight(s, ps *styles.Style, v string) {
	w := 400
	switch v {
	case "normal":
	case "bold":
		w = 700
	case "bolder":
		w = 700
		if ps != nil && ps.Font.Weight >= rich.Bold {
			w = 900
		}
	case "lighter":
		w = 300
	default:
		n, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		w = n
	}
	s.Font.Weight = rich.Weights(min(max((w+50)/100-1, 0), 8))
}

// fontFamily returns the font family for the given CSS font-family list,
// the first generic family in it, or one for a common font name.
func fontFamily(v string) rich.Family {
	for _, f := range strings.Split(v, ",") {
		f = strings.ToLower(strings.Trim(strings.TrimSpace(f), `"'`))
		switch f {
		case "monospace", "courier", "courier new", "consolas", "menlo", "monaco", "roboto mono", "ui-monospace", "sfmono-regular":
			return rich.Monospace
		case "serif", "times", "times new roman", "georgia", "garamond", "ui-serif":
			return rich.Serif
		case "sans-serif", "arial", "helvetica", "helvetica neue", "verdana", "tahoma", "system-ui", "-apple-system", "segoe ui", "roboto", "noto sans", "ui-sans-serif":
			return rich.SansSerif
		case "cursive":
			return rich.Cursive
		case "fantasy":
			return rich.Fantasy
		}
	}
	return rich.SansSerif
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
		case len(lf) > 0 && (lf[0] >= '0' && lf[0] <= '9' || lf[0] == '.') || cssFontSizes[lf] > 0:
			s.Font.Slant = slant
			s.Font.Weight = rich.Normal
			if weight != "" {
				setFontWeight(s, ps, weight)
			}
			size, lh, _ := strings.Cut(lf, "/")
			setFontSize(st.env, s, ps, size)
			st.lineHeight = "normal"
			if lh != "" {
				st.lineHeight = lh
			}
			family := strings.Join(fs[i+1:], " ")
			s.Font.Family = fontFamily(family)
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
	} else if px, ok := fontSizeKeywords[v]; ok {
		c.rootFont = px
	}
	return c.rootFont
}

// fontSizeKeywords are the font sizes of the absolute size keywords, in px.
var fontSizeKeywords = map[string]float32{"xx-small": 9, "x-small": 10, "small": 13, "medium": 16,
	"large": 18, "x-large": 24, "xx-large": 32, "xxx-large": 48}
