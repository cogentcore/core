// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// ToCSS converts the given [Style] object to a semicolon-separated CSS string.
// It is not guaranteed to be fully complete or accurate. It also takes the kebab-case
// ID name of the associated widget and the resultant html element name for context.
func ToCSS(s *Style, idName, htmlName string) string {
	parts := []string{}
	add := func(key, value string) {
		if value == "" || value == "0" || value == "0px" || value == "0dot" {
			return
		}
		parts = append(parts, key+":"+value)
	}
	length := func(v units.Value) string { return ToCSSLength(s, v) }
	// addAlign adds an alignment that is not the default (auto or start).
	addAlign := func(key string, a Aligns) {
		if a != Auto && a != Start {
			add(key, a.String())
		}
	}

	add("color", colorToCSS(s.Color))
	add("background", colorToCSS(s.Background))
	if htmlName == "svg" {
		add("stroke", colorToCSS(s.Color))
		add("fill", colorToCSS(s.Color))
	}
	if idName != "text" { // text does not have these layout properties
		if s.Is(states.Invisible) {
			add("display", "none")
		} else {
			add("display", s.Display.String())
		}
		add("flex-direction", s.Direction.String())
		add("flex-grow", fmt.Sprintf("%g", s.Grow.Y))
		add("justify-content", s.Justify.Content.String())
		add("align-items", s.Align.Items.String())
		if s.Display == Grid && s.Columns > 0 {
			add("grid-template-columns", fmt.Sprintf("repeat(%d,auto)", s.Columns))
		}
		add("row-gap", length(s.Gap.Y))
		add("column-gap", length(s.Gap.X))
		if s.Wrap {
			add("flex-wrap", "wrap")
		}
		if s.Shrink.X > 0 && s.Shrink.X != 1 { // 0 is a factor of 1, as in CSS
			add("flex-shrink", fmt.Sprintf("%g", s.Shrink.X))
		}
		if s.FlexBasis.Value > 0 {
			add("flex-basis", length(s.FlexBasis))
		}
		addAlign("align-content", s.Align.Content)
		addAlign("justify-items", s.Justify.Items)
		addAlign("justify-self", s.Justify.Self)
		if s.Overflow.X != OverflowVisible {
			add("overflow-x", s.Overflow.X.String())
		}
		if s.Overflow.Y != OverflowVisible {
			add("overflow-y", s.Overflow.Y.String())
		}
	}
	add("box-sizing", s.BoxSizing.String())
	if s.AspectRatio > 0 {
		add("aspect-ratio", fmt.Sprintf("%g", s.AspectRatio))
	}
	if s.ColSpan > 1 {
		add("grid-column", fmt.Sprintf("span %d", s.ColSpan))
	}
	if s.RowSpan > 1 {
		add("grid-row", fmt.Sprintf("span %d", s.RowSpan))
	}

	// The position of the widget, out of the layout or moved in it.
	if s.Position != PositionStatic {
		add("position", s.Position.String())
		set, inset := s.InsetSet, s.Inset
		for i, v := range []units.Value{inset.Top, inset.Right, inset.Bottom, inset.Left} {
			if []bool{set.Top, set.Right, set.Bottom, set.Left}[i] {
				add(cssSides[i], length(v))
			}
		}
		if s.ZIndex != 0 {
			add("z-index", strconv.Itoa(s.ZIndex))
		}
	}
	if s.Float != FloatNone {
		add("float", s.Float.String())
	}
	if s.Clear != ClearNone {
		add("clear", s.Clear.String())
	}
	var transform []string
	if s.Translate.X.Value != 0 || s.Translate.Y.Value != 0 {
		transform = append(transform, "translate("+length(s.Translate.X)+","+length(s.Translate.Y)+")")
	}
	if s.Rotate != 0 {
		transform = append(transform, fmt.Sprintf("rotate(%gdeg)", s.Rotate))
	}
	add("transform", strings.Join(transform, " "))
	add("min-width", length(s.Min.X))
	add("min-height", length(s.Min.Y))
	add("max-width", length(s.Max.X))
	add("max-height", length(s.Max.Y))
	if s.Grow == (math32.Vector2{}) {
		add("width", length(s.Min.X))
		add("height", length(s.Min.Y))
	}
	add("padding-top", length(s.Padding.Top))
	add("padding-right", length(s.Padding.Right))
	add("padding-bottom", length(s.Padding.Bottom))
	add("padding-left", length(s.Padding.Left))
	add("margin-top", length(s.Margin.Top))
	add("margin-right", length(s.Margin.Right))
	add("margin-bottom", length(s.Margin.Bottom))
	add("margin-left", length(s.Margin.Left))
	if s.Font.Size.Value != 16 || s.Font.Size.Unit != units.UnitDp {
		add("font-size", length(s.Font.Size))
	}
	// The font family of the page is the default sans-serif one.
	switch {
	case s.Font.Family == rich.Custom && s.Font.CustomFont != "":
		add("font-family", strconv.Quote(string(s.Font.CustomFont)))
	case s.Font.Family != rich.SansSerif && s.Font.Family != rich.Custom:
		add("font-family", s.Font.Family.String())
	}
	// A font of zero values (thin and ultra condensed) is not set, as for
	// widgets that only have the font of their rich text. The weight is
	// written even when it is normal, as for a heading, which is bold in
	// CSS; a stretch of zero is not set.
	if s.Font.Weight != rich.Thin || s.Font.Stretch != rich.UltraCondensed {
		add("font-weight", strconv.Itoa((int(s.Font.Weight)+1)*100)) // thin is 100
	}
	if s.Font.Stretch != rich.StretchNormal && s.Font.Stretch != rich.UltraCondensed {
		add("font-stretch", s.Font.Stretch.String())
	}
	if s.Font.Slant == rich.Italic {
		add("font-style", "italic")
	}
	add("text-decoration", decorationToCSS(s.Font.Decoration))
	add("line-height", fmt.Sprintf("%g", s.Text.LineHeight))
	add("text-align", s.Text.Align.String())
	switch s.Text.WhiteSpace {
	case text.WrapNever:
		add("white-space", "nowrap")
	case text.WhiteSpacePre:
		add("white-space", "pre")
	case text.WhiteSpacePreWrap:
		add("white-space", "pre-wrap")
	}
	add("text-indent", length(s.Text.Indent))
	if s.Text.TabSize > 0 && s.Text.TabSize != 4 {
		add("tab-size", strconv.Itoa(s.Text.TabSize))
	}
	if htmlName == "td" || htmlName == "th" { // as the valign of a cell
		switch s.Text.AlignV {
		case text.Center:
			add("vertical-align", "middle")
		case text.End:
			add("vertical-align", "bottom")
		}
	}

	// The border, shadows, and opacity of the box.
	bd := &s.Border
	widths := []units.Value{bd.Width.Top, bd.Width.Right, bd.Width.Bottom, bd.Width.Left}
	bstyles := []BorderStyles{bd.Style.Top, bd.Style.Right, bd.Style.Bottom, bd.Style.Left}
	bcolors := []image.Image{bd.Color.Top, bd.Color.Right, bd.Color.Bottom, bd.Color.Left}
	for i, w := range widths {
		if w.Value > 0 {
			add("border-"+cssSides[i], length(w)+" "+bstyles[i].String()+" "+colorToCSSOr(bcolors[i], "currentColor"))
		}
	}
	// The radii are of the corners: top-left, top-right, bottom-right, bottom-left.
	r := []string{length(bd.Radius.Top), length(bd.Radius.Right), length(bd.Radius.Bottom), length(bd.Radius.Left)}
	if r[0] == r[1] && r[0] == r[2] && r[0] == r[3] {
		add("border-radius", r[0])
	} else {
		add("border-radius", strings.Join(r, " "))
	}
	var shadows []string
	for _, sh := range s.BoxShadow {
		if !sh.HasShadow() {
			continue
		}
		css := length(sh.OffsetX) + " " + length(sh.OffsetY) + " " + length(sh.Blur) + " " + length(sh.Spread) + " " + colorToCSSOr(sh.Color, "currentColor")
		if sh.Inset {
			css = "inset " + css
		}
		shadows = append(shadows, css)
	}
	add("box-shadow", strings.Join(shadows, ","))
	if s.Opacity > 0 && s.Opacity < 1 {
		add("opacity", fmt.Sprintf("%g", s.Opacity))
	}

	return strings.Join(parts, ";")
}

// ToCSSLength returns the CSS of the given length of the given [Style], as
// in [ToCSS]: in px for a length in dots, which CSS does not have (as
// htmlcore sets), using the unit context of the style, or as
// [units.Value.StringCSSDp] otherwise.
func ToCSSLength(s *Style, v units.Value) string {
	if v.Unit == units.UnitDot {
		px := math32.Round(s.UnitContext.FromDots(v.Value, units.UnitDp)*100) / 100
		return strconv.FormatFloat(float64(px), 'g', -1, 32) + "px"
	}
	return v.StringCSSDp()
}

// cssSides are the names of the sides in CSS, in the order of [sides.Sides].
var cssSides = []string{"top", "right", "bottom", "left"}

// decorationToCSS returns the text-decoration of the given decorations.
func decorationToCSS(d rich.Decorations) string {
	var parts []string
	for _, f := range []rich.Decorations{rich.Underline, rich.Overline, rich.LineThrough} {
		if d.HasFlag(f) {
			parts = append(parts, f.String())
		}
	}
	if d.HasFlag(rich.DottedUnderline) {
		parts = append(parts, "underline dotted")
	}
	return strings.Join(parts, " ")
}

// colorToCSSOr returns the CSS of the given color, or def if it is
// already the default (as for the color of a border, which is the text
// color in CSS).
func colorToCSSOr(c image.Image, def string) string {
	if css := colorToCSS(c); css != "" {
		return css
	}
	if c == nil {
		return def
	}
	return colors.AsHex(colors.ToUniform(c))
}

// linearToCSS returns the CSS linear-gradient of a linear gradient whose
// coordinates are relative to the box of the element.
func linearToCSS(g *gradient.Linear) string {
	d := g.End.Sub(g.Start)
	// The angle of CSS is clockwise from the top (to top is 0deg).
	angle := math32.RadToDeg(math32.Atan2(d.X, -d.Y))
	stops := []string{fmt.Sprintf("%gdeg", math32.Round(angle))}
	for _, st := range g.Stops {
		c := st.Color
		if st.Opacity < 1 {
			c = colors.ApplyOpacity(c, st.Opacity)
		}
		stops = append(stops, fmt.Sprintf("%s %g%%", colors.AsHex(c), math32.Round(st.Pos*1000)/10))
	}
	return "linear-gradient(" + strings.Join(stops, ",") + ")"
}

func colorToCSS(c image.Image) string {
	switch c {
	case nil:
		return ""
	case colors.Scheme.Primary.Base:
		return "var(--primary-color)"
	case colors.Scheme.Primary.On:
		return "var(--primary-on-color)"
	case colors.Scheme.Secondary.Container:
		return "var(--secondary-container-color)"
	case colors.Scheme.Secondary.OnContainer:
		return "var(--secondary-on-container-color)"
	case colors.Scheme.Surface, colors.Scheme.OnSurface:
		return "" // already default
	case colors.Scheme.SurfaceContainer, colors.Scheme.SurfaceContainerLowest, colors.Scheme.SurfaceContainerLow, colors.Scheme.SurfaceContainerHigh, colors.Scheme.SurfaceContainerHighest:
		return "var(--surface-container-color)" // all of them are close enough for this
	default:
		if g, ok := c.(*gradient.Linear); ok && g.Units == gradient.ObjectBoundingBox && len(g.Stops) > 1 {
			return linearToCSS(g)
		}
		return colors.AsHex(colors.ToUniform(c))
	}
}
