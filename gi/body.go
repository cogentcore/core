// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/girl/styles"
)

// Body holds the primary content of a Scene
type Body struct { //goki:no-new
	Frame

	// title of the Scene, also used for window title where relevant
	Title string
}

// NewBody creates a new Body that will serve as the content of a Scene
// (e.g., a Window, Dialog, etc).  Body forms the central region
// of a Scene, and has OverflowAuto scrollbars by default.
// It will create its own parent Scene at this point, and has wrapper
// functions to transparently manage everything that the Scene
// typically manages during configuration, so you can usually avoid
// having to access the Scene directly.
func NewBody(name ...string) *Body {
	bd := &Body{}
	nm := AppName()
	if len(name) > 0 {
		nm = name[0]
	}
	bd.InitName(bd, nm)
	bd.Sc = NewBodyScene(bd)
	return bd
}

func (bd *Body) OnInit() {
	bd.Frame.OnInit()
	bd.BodyStyles()
}

func (bd *Body) BodyStyles() {
	bd.Style(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
}

// AddTitle adds a Label with given title, and sets the Title text
// which will be used by the Scene etc.
func (bd *Body) AddTitle(title string) *Body {
	bd.Title = title
	bd.Sc.Nm = title
	NewLabel(bd, "title").SetText(title).SetType(LabelHeadlineSmall)
	return bd
}

// AddText adds the given supporting text Label, typically added
// after a title.
func (bd *Body) AddText(text string) *Body {
	NewLabel(bd, "text").SetText(text).
		SetType(LabelBodyMedium).Style(func(s *styles.Style) {
		s.Color = colors.Scheme.OnSurfaceVariant
	})
	return bd
}
