// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

func TestDialogMessage(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(10))
	})
	b.AssertRenderScreen(t, "dialog/message", func() {
		MessageDialog(b, "Something happened", "Message")
	})
}
