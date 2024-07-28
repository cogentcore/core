// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles/units"
)

// ToCSS converts the given [Style] object to a semicolon-separated CSS string.
// It is not guaranteed to be fully complete or accurate.
func ToCSS(s *Style) string {
	parts := []string{}
	add := func(key, value string) {
		parts = append(parts, key+":"+value)
	}

	add("color", colors.AsHex(colors.ToUniform(s.Color)))
	if s.Background != nil {
		add("background", colors.AsHex(colors.ToUniform(s.Background)))
	}
	if s.Font.Size.Value != 16 || s.Font.Size.Unit != units.UnitDp {
		add("font-size", s.Font.Size.StringCSS())
	}
	if s.Font.Family != "Roboto" {
		add("font-family", s.Font.Family)
	}
	if s.Border.Width.Top.Value > 0 {
		add("border-style", s.Border.Style.Top.String())
		add("border-width", s.Border.Width.Top.StringCSS())
		add("border-color", colors.AsHex(colors.ToUniform(s.Border.Color.Top)))
	}
	if s.Border.Radius.Top.Value > 0 {
		add("border-radius", s.Border.Radius.Top.StringCSS())
	}

	return strings.Join(parts, ";")
}
