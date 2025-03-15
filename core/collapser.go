// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
)

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
	cl.Summary.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Align.Content = styles.Center
		s.Align.Items = styles.Center
		s.Gap.X.Em(0.1)
	})

	toggle := NewSwitch(cl.Summary).SetType(SwitchCheckbox).SetIconOn(icons.KeyboardArrowDown).SetIconOff(icons.KeyboardArrowRight)
	toggle.SetName("toggle")
	Bind(&cl.Open, toggle)
	toggle.Styler(func(s *styles.Style) {
		s.Color = colors.Scheme.Primary.Base
		s.Padding.Zero()
	})
	toggle.OnChange(func(e events.Event) {
		cl.Update()
	})

	cl.Details = NewFrame(cl)
	cl.Details.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Direction = styles.Column
	})
	cl.Details.Updater(func() {
		cl.Details.SetState(!cl.Open, states.Invisible)
	})
}
