// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pages

import (
	"embed"
	"testing"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

//go:embed examples/basic/content
var content embed.FS

func TestPage(t *testing.T) {
	b := core.NewBody("Pages Example")
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(50))
	})
	NewPage(b).SetSource(fsx.Sub(content, "examples/basic/content"))
	b.AssertRender(t, "basic")
}
