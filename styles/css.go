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
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
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
		add("gap", length(s.Gap.X))
	}
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
	// todo:
	// if s.Font.Family != "" && s.Font.Family != "Roboto" {
	// 	ff := s.Font.Family
	// 	if strings.HasSuffix(ff, "Mono") {
	// 		ff += ", monospace"
	// 	} else {
	// 		ff += ", sans-serif"
	// 	}
	// 	add("font-family", ff)
	// }
	if s.Font.Weight == rich.Medium {
		add("font-weight", "500")
	} else {
		add("font-weight", s.Font.Weight.String())
	}
	add("line-height", fmt.Sprintf("%g", s.Text.LineHeight))
	add("text-align", s.Text.Align.String())
	if s.Border.Width.Top.Value > 0 {
		add("border-style", s.Border.Style.Top.String())
		add("border-width", length(s.Border.Width.Top))
		add("border-color", colorToCSS(s.Border.Color.Top))
	}
	add("border-radius", length(s.Border.Radius.Top))

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
		return colors.AsHex(colors.ToUniform(c))
	}
}
