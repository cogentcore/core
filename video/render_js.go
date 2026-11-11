// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package video

import (
	"image"
	"image/draw"

	"cogentcore.org/core/system/composer"
)

// videoSource implements [composer.Source] for core direct rendering:
// it places the video element of the video at the widget.
type videoSource struct {
	v *Video

	// rect is the box of the content of the widget, and visible is its
	// visible part, in the coordinates of the window.
	rect, visible image.Rectangle
}

func (vs *videoSource) Draw(c composer.Composer) {
	cw := c.(*composer.ComposerWeb)
	e := vs.v.element()
	cw.SetElementContext(vs.v.This, e)
	cw.SetElementBelow(vs.v.This) // (under the canvas, see video_js.go)
	cw.SetElementGeom(e, vs.rect.Min, vs.rect.Size())
	cw.SetElementClip(e, vs.rect, vs.visible)
}

// RenderSource returns the [composer.Source] for direct rendering. It is
// drawn while the video has media, even when it is not visible (clipped to
// nothing then), as the composer removes the element of a source that is
// not drawn, and the browser pauses a video element that is removed.
func (v *Video) RenderSource(op draw.Op) composer.Source {
	if v.Media == nil {
		return nil
	}
	off := v.Scene.SceneGeom.Pos
	min := v.Geom.Pos.Content.ToPoint()
	rect := image.Rectangle{Min: min, Max: min.Add(v.Geom.Size.Actual.Content.ToPoint())}.Add(off)
	var visible image.Rectangle
	if v.IsVisible() {
		visible = v.Geom.ContentBBox.Add(off)
	}
	return &videoSource{v: v, rect: rect, visible: visible}
}
