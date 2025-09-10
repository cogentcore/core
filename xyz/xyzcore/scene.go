// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package xyzcore provides a GUI view for a 3D xyz scene.
package xyzcore

//go:generate core generate

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/xyz"
)

// Scene is a core.Widget that manages a xyz.Scene,
// providing the basic rendering logic for the 3D scene
// in the 2D core GUI context.
type Scene struct {
	core.WidgetBase

	// XYZ is the 3D xyz.Scene
	XYZ *xyz.Scene `set:"-"`

	// how to deal with selection / manipulation events
	SelectionMode SelectionModes

	// currently selected node
	CurrentSelected xyz.Node `copier:"-" json:"-" xml:"-" display:"-"`

	// currently selected manipulation control point
	CurrentManipPoint *ManipPoint `copier:"-" json:"-" xml:"-" display:"-"`

	// parameters for selection / manipulation box
	SelectionParams SelectionParams `display:"inline"`
}

func (sw *Scene) Init() {
	sw.WidgetBase.Init()
	sw.XYZ = xyz.NewScene()
	sw.SelectionParams.Defaults()
	sw.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable, abilities.Focusable, abilities.Activatable, abilities.Slideable, abilities.LongHoverable, abilities.DoubleClickable)
		s.Grow.Set(1, 1)
		s.Min.Set(units.Em(2))
	})

	sw.On(events.Scroll, func(e events.Event) {
		pos := sw.Geom.ContentBBox.Min
		e.SetLocalOff(e.LocalOff().Add(pos))
		sw.XYZ.MouseScrollEvent(e.(*events.MouseScroll))
		sw.NeedsRender()
	})
	sw.On(events.KeyChord, func(e events.Event) {
		sw.XYZ.KeyChordEvent(e)
		sw.NeedsRender()
	})
	sw.handleSlideEvents()
	sw.handleSelectEvents()

	sw.Updater(func() {
		sz := sw.Geom.ContentBBox.Size()
		if sz.X <= 0 || sz.Y <= 0 {
			return
		}
		sw.XYZ.Geom.Size = sz

		doRebuild := sw.NeedsRebuild() // settings-driven full rebuild
		if sw.XYZ.Frame != nil {
			cursz := sw.XYZ.Frame.Render().Format.Size
			if cursz == sz && !doRebuild {
				sw.XYZ.SetNeedsRender()
				return
			}
		} else {
			doRebuild = false // will be done automatically b/c Frame == nil
		}
		sw.configFrame(sz)
		if doRebuild {
			sw.XYZ.Rebuild()
		}
		sw.NeedsRender()
	})
}

func (sw *Scene) OnAdd() {
	sw.WidgetBase.OnAdd()
	sw.Scene.AddDirectRender(sw)
}

func (sw *Scene) Destroy() {
	sw.Scene.DeleteDirectRender(sw)
	sw.XYZ.Destroy()
	sw.WidgetBase.Destroy()
}

// SceneXYZ returns the xyz.Scene
func (sw *Scene) SceneXYZ() *xyz.Scene {
	return sw.XYZ
}

func (sw *Scene) Render() {
	sw.UpdateWidget() // Note: this is indeed essential here -- doesn't work without it.
	if sw.XYZ.Frame == nil {
		return
	}
	sw.XYZ.DoUpdate()
}
