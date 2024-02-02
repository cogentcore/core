// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// For https://github.com/cogentcore/core/issues/614
func TestRenderOneSideBorder(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Test")
	NewBox(b).Style(func(s *styles.Style) {
		s.Min.Set(units.Dp(100))
		s.Border.Width.Bottom.Dp(10)
		s.Border.Color.Bottom = colors.Scheme.Outline
		s.Background = colors.C(colors.Scheme.SurfaceContainerHigh)
	})
	b.AssertRender(t, filepath.Join("render", "one-side-border"))
}
