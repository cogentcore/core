// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/styles"
)

// Body holds the primary content of a Scene
type Body struct { //core:no-new
	Frame

	// title of the Body, also used for window title where relevant
	Title string
}

// NewBody creates a new Body that will serve as the content of a Scene
// (e.g., a Window, Dialog, etc).  Body forms the central region
// of a Scene, and has OverflowAuto scrollbars by default.
// It will create its own parent Scene at this point, and has wrapper
// functions to transparently manage everything that the Scene
// typically manages during configuration, so you can usually avoid
// having to access the Scene directly. If a name is given and
// the name of [TheApp] is unset, it sets it to the given name.
func NewBody(name ...string) *Body {
	bd := &Body{}
	nm := "body"
	if len(name) > 0 {
		nm = name[0]
		if TheApp.Name() == "" {
			TheApp.SetName(nm)
		}
	}
	if AppearanceSettings.Zoom == 0 {
		// we load the settings in NewBody so that people can
		// add their own settings to AllSettings first
		grr.Log(LoadAllSettings())
	}
	bd.InitName(bd, nm)
	bd.Title = nm
	bd.Scene = NewBodyScene(bd)
	return bd
}

func (b *Body) OnInit() {
	b.Frame.OnInit()
	b.SetStyles()
}

func (b *Body) SetStyles() {
	b.Style(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
}

// AddTitle adds a Label with given title, and sets the Title text
// which will be used by the Scene etc.
func (b *Body) AddTitle(title string) *Body {
	b.Title = title
	b.Scene.Nm = title
	NewLabel(b, "title").SetText(title).SetType(LabelHeadlineSmall)
	return b
}

// AddText adds the given supporting text Label, typically added
// after a title.
func (b *Body) AddText(text string) *Body {
	NewLabel(b, "text").SetText(text).
		SetType(LabelBodyMedium).Style(func(s *styles.Style) {
		s.Color = colors.Scheme.OnSurfaceVariant
	})
	return b
}

// SetData sets the Body's [Scene.Data].
func (b *Body) SetData(data any) *Body {
	b.Scene.SetData(data)
	return b
}
