// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/glop/sentence"
	"cogentcore.org/core/styles"
)

// Body holds the primary content of a Scene
type Body struct { //goki:no-new
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
// having to access the Scene directly.
func NewBody(name ...string) *Body {
	bd := &Body{}
	nm := ""
	if len(name) > 0 {
		nm = name[0]
	}
	bd.InitName(bd, nm)
	bd.Title = sentence.Case(nm)
	bd.Sc = NewBodyScene(bd)
	return bd
}

func (bd *Body) OnInit() {
	bd.Frame.OnInit()
	bd.SetStyles()
}

func (bd *Body) SetStyles() {
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

// SetApp sets the App of the Body's Scene
func (bd *Body) SetApp(app *App) *Body {
	bd.Sc.App = app
	bd.Nm = app.Name
	bd.Title = sentence.Case(bd.Nm)
	return bd
}

// SetData sets the Body's [Scene.Data].
func (bd *Body) SetData(data any) *Body {
	bd.Sc.SetData(data)
	return bd
}
