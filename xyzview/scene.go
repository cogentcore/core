// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package xyzview provides a GUI view for a 3D xyz scene.
package xyzview

//go:generate core generate

import (
	"image"
	"image/draw"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/units"
	"cogentcore.org/core/vgpu"
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
	CurrentSelected xyz.Node `copier:"-" json:"-" xml:"-" view:"-"`

	// currently selected manipulation control point
	CurrentManipPoint *ManipPoint `copier:"-" json:"-" xml:"-" view:"-"`

	// parameters for selection / manipulation box
	SelectionParams SelectionParams `view:"inline"`
}

func (sw *Scene) OnInit() {
	sw.XYZ = xyz.NewScene("Scene")
	sw.XYZ.Defaults()
	sw.SelectionParams.Defaults()
	sw.WidgetBase.OnInit()
	sw.HandleEvents()
	sw.SetStyles()
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

func (sw *Scene) SetStyles() {
	sw.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable, abilities.Focusable, abilities.Activatable, abilities.Slideable, abilities.LongHoverable, abilities.DoubleClickable)
		s.Grow.Set(1, 1)
		s.Min.Set(units.Em(20))
	})
}

func (sw *Scene) HandleEvents() {
	sw.On(events.Scroll, func(e events.Event) {
		// pos := sw.Geom.ContentBBox.Min
		// fmt.Println("loc off:", e.LocalOff(), "pos:", pos, "e pos:", e.WindowPos())
		// e.SetLocalOff(e.LocalOff().Add(pos))
		sw.XYZ.MouseScrollEvent(e.(*events.MouseScroll))
		sw.NeedsRender()
	})
	sw.On(events.KeyChord, func(e events.Event) {
		sw.XYZ.KeyChordEvent(e)
		sw.NeedsRender()
	})
	sw.HandleSlideEvents()
	sw.HandleSelectEvents()
}

func (sw *Scene) Config() {
	sw.ConfigFrame()
}

// ConfigFrame configures the framebuffer for GPU rendering,
// using the RenderWindow GPU and Device.
func (sw *Scene) ConfigFrame() {
	zp := image.Point{}
	sz := sw.Geom.Size.Actual.Content.ToPointFloor()
	if sz == zp {
		return
	}
	sw.XYZ.Geom.Size = sz

	doConfig := false
	if sw.XYZ.Frame != nil {
		cursz := sw.XYZ.Frame.Format.Size
		if cursz == sz {
			return
		}
	} else {
		doConfig = true
	}

	win := sw.WidgetBase.Scene.Events.RenderWindow()
	if win == nil {
		return
	}
	drw := win.SystemWindow.Drawer()
	system.TheApp.RunOnMain(func() {
		sw.XYZ.ConfigFrameFromSurface(drw.Surface().(*vgpu.Surface))
		if doConfig {
			sw.XYZ.Config()
		}
	})
	sw.XYZ.SetFlag(true, xyz.ScNeedsRender)
	sw.NeedsRender()
}

func (sw *Scene) Render() {
	sw.ConfigFrame() // no-op if all good
	if sw.XYZ.Frame == nil {
		return
	}
	if sw.XYZ.Is(xyz.ScNeedsConfig) {
		system.TheApp.RunOnMain(func() {
			sw.XYZ.Config()
		})
	}
	sw.XYZ.DoUpdate()
}

// DirectRenderImage uploads framebuffer image
func (sw *Scene) DirectRenderImage(drw system.Drawer, idx int) {
	if sw.XYZ.Frame == nil || !sw.IsVisible() {
		return
	}
	drw.SetFrameImage(idx, sw.XYZ.Frame.Frames[0])
}

// DirectRenderDraw draws the current image to RenderWindow drawer
func (sw *Scene) DirectRenderDraw(drw system.Drawer, idx int, flipY bool) {
	if !sw.IsVisible() {
		return
	}
	bb := sw.Geom.TotalBBox
	ibb := image.Rectangle{Max: bb.Size()}
	drw.Copy(idx, 0, bb.Min, ibb, draw.Src, flipY)
}
