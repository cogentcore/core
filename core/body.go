// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Body holds the primary content of a Scene
type Body struct { //core:no-new
	Frame

	// title of the Body, also used for window title where relevant
	Title string `set:"-"`
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
	bd := tree.New[*Body]()
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
		errors.Log(LoadAllSettings())
	}
	bd.SetName(nm)
	bd.Title = nm
	bd.Scene = NewBodyScene(bd)
	return bd
}

func (bd *Body) Init() {
	bd.Frame.Init()
	bd.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
}

// SetTitle sets the title in the Body, Scene, and Stage, RenderWindow, and title widget.
// This is the one place to change the title for everything.
func (bd *Body) SetTitle(title string) *Body {
	bd.Nm = title
	bd.Title = title
	bd.Scene.Nm = title
	if bd.Scene.Stage != nil {
		bd.Scene.Stage.Title = title
		win := bd.Scene.RenderWindow()
		if win != nil {
			win.SetName(title)
			win.SetTitle(title)
		}
	}
	if lb, ok := bd.ChildByName("body-title", 0).(*Text); ok {
		lb.SetText(title)
	}
	return bd
}

// AddTitle adds [Text] with the given title, and sets the Title text
// which will be used by the Scene etc.
func (bd *Body) AddTitle(title string) *Body {
	bd.SetTitle(title)
	NewText(bd).SetText(title).SetType(TextHeadlineSmall).SetName("body-title")
	return bd
}

// AddText adds the given supporting [Text], typically added
// after a title.
func (bd *Body) AddText(text string) *Body {
	NewText(bd).SetText(text).
		SetType(TextBodyMedium).Styler(func(s *styles.Style) {
		s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
	})
	return bd
}

// SetData sets the Body's [Scene.Data].
func (bd *Body) SetData(data any) *Body {
	bd.Scene.SetData(data)
	return bd
}
