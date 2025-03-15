// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "cogentcore.org/core/styles"

// Collapser is a widget that can be collapsed or expanded by a user.
// The [Collapser.Summary] is always visible, and the [Collapser.Details]
// are only visible when the [Collapser] is expanded with [Collapser.Open]
// equal to true.
//
// You can directly add any widgets to the [Collapser.Summary] and [Collapser.Details]
// by specifying one of them as the parent in calls to New{WidgetName}.
// Collapser is similar to HTML's <details> and <summary> tags.
type Collapser struct {
	Frame

	// Open is whether the collapser is currently expanded. It defaults to false.
	Open bool

	// Summary is the part of the collapser that is always visible.
	Summary *Frame `set:"-"`

	// Details is the part of the collapser that is only visible when
	// the collapser is expanded.
	Details *Frame `set:"-"`
}

func (cl *Collapser) Init() {
	cl.Frame.Init()

	cl.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 0)
	})
}

func (cl *Collapser) OnAdd() {
	cl.Frame.OnAdd()

	cl.Summary = NewFrame(cl)
	cl.Details = NewFrame(cl)
}
