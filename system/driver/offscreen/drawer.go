// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"image"

	"cogentcore.org/core/system"
)

// Drawer is the implementation of [system.Drawer] for the offscreen platform
type Drawer struct {
	system.DrawerBase

	Window *Window
}

// DestBounds returns the bounds of the render destination
func (dw *Drawer) DestBounds() image.Rectangle {
	rect := image.Rectangle{Max: dw.Window.PixelSize}
	if dw.Image.Bounds().Size() != dw.Window.PixelSize {
		dw.Image = image.NewRGBA(rect)
	}
	return rect
}

func (dw *Drawer) End() {} // no-op

// GetImage returns the rendered image. It is called through an interface
// in core.Body.AssertRenderWindow.
func (dw *Drawer) GetImage() *image.RGBA {
	return dw.Image
}
