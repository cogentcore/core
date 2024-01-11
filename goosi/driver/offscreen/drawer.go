// Copyright 2023 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"image"

	"goki.dev/goosi"
)

// Drawer is the implementation of [goosi.Drawer] for the offscreen platform
type Drawer struct {
	goosi.DrawerBase
}

// DestBounds returns the bounds of the render destination
func (dw *Drawer) DestBounds() image.Rectangle {
	return TheApp.Scrn.Geometry
}

// EndDraw ends image drawing rendering process on render target.
// This is the function that actually sends the image to the capture channel.
func (dw *Drawer) EndDraw() {
	if !goosi.NeedsCapture {
		return
	}
	goosi.CaptureImage <- dw.Image
}
