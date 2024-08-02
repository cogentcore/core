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
		add("columns", strconv.Itoa(s.Columns))
		add("gap", s.Gap.X.StringCSS())
	}
	add("min-width", s.Min.X.StringCSS())
	add("min-height", s.Min.Y.StringCSS())
	add("max-width", s.Max.X.StringCSS())
	add("max-height", s.Max.Y.StringCSS())
	if s.Grow == (math32.Vector2{}) {
		add("width", s.Min.X.StringCSS())
		add("height", s.Min.Y.StringCSS())
	}
	add("padding-top", s.Padding.Top.StringCSS())
	add("padding-right", s.Padding.Right.StringCSS())
	add("padding-bottom", s.Padding.Bottom.StringCSS())
	add("padding-left", s.Padding.Left.StringCSS())
	add("margin", s.Margin.Top.StringCSS())
	if s.Font.Size.Value != 16 || s.Font.Size.Unit != units.UnitDp {
		add("font-size", s.Font.Size.StringCSS())
	}
	if s.Font.Family != "" && s.Font.Family != "Roboto" {
		ff := s.Font.Family
		if strings.HasSuffix(ff, "Mono") {
			ff += ", monospace"
		} else {
			ff += ", sans-serif"
		}
		add("font-family", ff)
	}
	if s.Font.Weight == WeightMedium {
		add("font-weight", "500")
	} else {
		add("font-weight", s.Font.Weight.String())
	}
	add("line-height", s.Text.LineHeight.StringCSS())
	add("text-align", s.Text.Align.String())
	if s.Border.Width.Top.Value > 0 {
		add("border-style", s.Border.Style.Top.String())
		add("border-width", s.Border.Width.Top.StringCSS())
		add("border-color", colorToCSS(s.Border.Color.Top))
	}
	add("border-radius", s.Border.Radius.Top.StringCSS())

	return strings.Join(parts, ";")
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
	case colors.Scheme.Surface, colors.Scheme.OnSurface, colors.Scheme.Background, colors.Scheme.OnBackground:
		return "" // already default
	case colors.Scheme.SurfaceContainer, colors.Scheme.SurfaceContainerLowest, colors.Scheme.SurfaceContainerLow, colors.Scheme.SurfaceContainerHigh, colors.Scheme.SurfaceContainerHighest:
		return "var(--surface-container-color)" // all of them are close enough for this
	default:
		return colors.AsHex(colors.ToUniform(c))
	}
}
