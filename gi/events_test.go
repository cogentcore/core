// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"

	"cogentcore.org/core/icons"
)

func TestWidgetPrev(t *testing.T) {
	b := NewBody()
	NewTextField(b, "tf1").AddClearButton()
	NewTextField(b, "tf2").SetLeadingIcon(icons.Search)
	lt := NewTextField(b, "tf3")
	b.ConfigTree()

	paths := []string{
		"/body/tf2.parts/lead-icon.parts/icon",
		"/body/tf2.parts/lead-icon",
		"/body/tf2",
		"/body/tf1.parts/trail-icon.parts/icon",
		"/body/tf1.parts/trail-icon",
		"/body/tf1.parts/trail-icon-str",
		"/body/tf1",
		"/body",
	}
	i := 0
	WidgetPrevFunc(lt, func(w Widget) bool {
		// fmt.Println(w)
		p := w.Path()
		tp := paths[i]
		if p != tp {
			t.Errorf("path != target: path: %s != target: %s\n", p, tp)
		}
		i++
		return false
	})
}
