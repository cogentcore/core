// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package video

import (
	"image"
	"image/color"
	"image/draw"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/paint/renderers/rasterx"
	"cogentcore.org/core/system/composer"
)

// videoSource implements [composer.Source] for core direct rendering:
// it draws the frame in the box of the video, fitted to it with black
// around it, as in players, and the region of the controls of the scene
// over it, as the region of the video in the scene is transparent
// (see [Video.Render]).
type videoSource struct {
	// box is the visible part of the content of the video, in the
	// coordinates of the window, which is filled with black.
	box image.Rectangle

	// destBBox and srcBBox are the regions of the window and the frame
	// in which the visible part of the frame is drawn.
	destBBox, srcBBox image.Rectangle

	rotation  float32
	unchanged bool
	frame     *image.RGBA

	// scene is the image of the scene, and controls is the region of the
	// controls in it, which is drawn at controlsPos in the window.
	scene       image.Image
	controls    image.Rectangle
	controlsPos image.Point
}

func (vs *videoSource) Draw(c composer.Composer) {
	cd := c.(*composer.ComposerDrawer)
	cd.Drawer.Copy(image.Point{}, colors.Uniform(color.Black), vs.box, draw.Src, composer.Unchanged) // (a fill of the box)
	if !vs.destBBox.Empty() {
		cd.Drawer.Scale(vs.destBBox, vs.frame, vs.srcBBox, vs.rotation, draw.Src, vs.unchanged)
	}
	if vs.scene != nil && !vs.controls.Empty() {
		// (the image of the scene is already on the GPU, drawn before)
		cd.Drawer.Copy(vs.controlsPos, vs.scene, vs.controls, draw.Over, composer.Unchanged)
	}
}

// RenderSource returns the [composer.Source] for direct rendering.
func (v *Video) RenderSource(op draw.Op) composer.Source {
	frame, unchanged := v.CurrentFrame()
	if frame == nil {
		return nil
	}
	off := v.Scene.SceneGeom.Pos
	visible := v.Geom.ContentBBox
	if visible.Empty() {
		return nil
	}
	vs := &videoSource{box: visible.Add(off), rotation: v.Rotation, unchanged: unchanged, frame: frame}

	// The frame is fitted in the content of the video, keeping its
	// aspect (turned for a rotation of 90 degrees).
	pos := v.Geom.Pos.Content.ToPoint()
	content := image.Rectangle{Min: pos, Max: pos.Add(v.Geom.Size.Actual.Content.ToPoint())}
	fsz := frame.Bounds().Size()
	if r := int(v.Rotation) % 180; r == 90 || r == -90 {
		fsz.X, fsz.Y = fsz.Y, fsz.X
	}
	fit := fitRect(content, fsz)
	dest := fit.Intersect(visible)
	if v.Rotation == 0 && dest != fit && !dest.Empty() {
		// the part of the frame of the visible part of its region
		sx := float64(fsz.X) / float64(fit.Dx())
		sy := float64(fsz.Y) / float64(fit.Dy())
		vs.srcBBox = image.Rect(int(float64(dest.Min.X-fit.Min.X)*sx), int(float64(dest.Min.Y-fit.Min.Y)*sy),
			int(float64(dest.Max.X-fit.Min.X)*sx), int(float64(dest.Max.Y-fit.Min.Y)*sy))
		vs.destBBox = dest.Add(off)
	} else if dest == fit {
		vs.srcBBox, vs.destBBox = frame.Bounds(), fit.Add(off)
	}

	// The controls of the scene, over the frame.
	if c := v.controls; c != nil && c.IsVisible() {
		if rd, ok := v.Scene.Renderer.(*rasterx.Renderer); ok {
			vs.scene = rd.Image()
			vs.controls = c.Geom.TotalBBox.Intersect(visible)
			vs.controlsPos = vs.controls.Min.Add(off)
		}
	}
	return vs
}

// fitRect returns the rectangle of the given size fitted in the given
// rectangle, keeping its aspect, centered in it.
func fitRect(r image.Rectangle, size image.Point) image.Rectangle {
	if size.X <= 0 || size.Y <= 0 || r.Empty() {
		return image.Rectangle{}
	}
	w, h := r.Dx(), r.Dx()*size.Y/size.X
	if h > r.Dy() {
		w, h = r.Dy()*size.X/size.Y, r.Dy()
	}
	min := r.Min.Add(image.Pt((r.Dx()-w)/2, (r.Dy()-h)/2))
	return image.Rectangle{Min: min, Max: min.Add(image.Pt(w, h))}
}
