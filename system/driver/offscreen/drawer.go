// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"image"

	"cogentcore.org/core/system/composer"
)

// Drawer is the implementation of [system.Drawer] for the offscreen platform
type Drawer struct {
	composer.DrawerBase

	Window *Window
}

func (dw *Drawer) Start() {
	rect := image.Rectangle{Max: dw.Window.PixelSize}
	if dw.Image == nil || dw.Image.Rect != rect {
		dw.Image = image.NewRGBA(rect)
	}
	dw.DrawerBase.Start()
}

func (dw *Drawer) End() {} // no-op

func (dw *Drawer) Redraw() {} // no-op

// GetImage returns the rendered image. It is called through an interface
// in core.Body.AssertRender.
func (dw *Drawer) GetImage() *image.RGBA {
	return dw.Image
}
