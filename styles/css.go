// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"strings"

	"cogentcore.org/core/colors"
)

// ToCSS converts the given [Style] object to a semicolon-separated CSS string.
// It is not guaranteed to be fully complete or accurate.
func ToCSS(s *Style) string {
	parts := []string{}
	add := func(key, value string) {
		parts = append(parts, key+":"+value)
	}

	add("color", colors.AsHex(colors.ToUniform(s.Color)))
	add("background", colors.AsHex(colors.ToUniform(s.Background)))
	add("font-size", s.Font.Size.StringCSS())
	add("border-style", s.Border.Style.Top.String())
	add("border-width", s.Border.Width.Top.StringCSS())
	add("border-radius", s.Border.Radius.Top.StringCSS())
	add("border-offset", s.Border.Offset.Top.StringCSS())
	add("border-color", colors.AsHex(colors.ToUniform(s.Border.Color.Top)))

	return strings.Join(parts, ";")
}
