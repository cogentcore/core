// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Body holds the primary content of a [Scene].
// It is the main container for app content.
type Body struct { //core:no-new
	Frame

	// Title is the title of the body, which is also
	// used for the window title where relevant.
	Title string `set:"-"`
}

// NewBody creates a new [Body] that will serve as the content of a [Scene]
// (e.g., a Window, Dialog, etc). [Body] forms the central region
// of a [Scene], and has [styles.OverflowAuto] scrollbars by default.
// It will create its own parent [Scene] at this point, and has wrapper
// functions to transparently manage everything that the [Scene]
// typically manages during configuration, so you can usually avoid
// having to access the [Scene] directly. If a name is given, it will
// be used for the name of the window, and a title widget will be created
// with that text if [Stage.DisplayTitle] is true. Also, if the name of
// [TheApp] is unset, it sets it to the given name.
func NewBody(name ...string) *Body {
	bd := tree.New[Body]()
	nm := "body"
	if len(name) > 0 {
		nm = name[0]
	}
	if TheApp.Name() == "" {
		if len(name) == 0 {
			nm = "Cogent Core" // first one is called Cogent Core by default
		}
		TheApp.SetName(nm)
	}
	if AppearanceSettings.Zoom == 0 {
		// we load the settings in NewBody so that people can
		// add their own settings to AllSettings first
		errors.Log(LoadAllSettings())
	}
	bd.SetName(nm)
	bd.Title = nm
	bd.Scene = newBodyScene(bd)
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

// SetTitle sets the title in the [Body], [Scene], [Stage], [renderWindow],
// and title widget. This is the one place to change the title for everything.
func (bd *Body) SetTitle(title string) *Body {
	bd.Name = title
	bd.Title = title
	bd.Scene.Name = title
	if bd.Scene.Stage != nil {
		bd.Scene.Stage.Title = title
		win := bd.Scene.RenderWindow()
		if win != nil {
			win.setName(title)
			win.setTitle(title)
		}
	}
	// title widget is contained within the top bar
	if tb, ok := bd.Scene.ChildByName("top-bar").(Widget); ok {
		tb.AsWidget().Update()
	}
	return bd
}

// SetData sets the [Body]'s [Scene.Data].
func (bd *Body) SetData(data any) *Body {
	bd.Scene.SetData(data)
	return bd
}
