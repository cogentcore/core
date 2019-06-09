// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/goki/ki/ints"
)

// A Sprite is just an image (with optional background) that can be drawn onto
// the OverTex overlay texture of a window.  Sprites are used for cursors
// and for dynamic editing / interactive GUI elements (e.g., drag-n-drop elments)
type Sprite struct {
	On     bool        `desc:"whether this sprite is active now or not"`
	Name   string      `desc:"unique name of sprite"`
	Geom   Geom2DInt   `desc:"position and size of the image within the overlay window texture"`
	Pixels *image.RGBA `desc:"pixels to render -- should be same size as Geom.Size"`
	Bg     *image.RGBA `desc:"optional background image which is rendered first before IMage"`
}

// Sprites is a map of named Sprite elements
type Sprites map[string]*Sprite

// Resize resizes sprite to given size
func (sp *Sprite) Resize(nwsz image.Point) {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return
	}
	sp.Geom.Size = nwsz // always make sure
	if sp.Pixels != nil && sp.Pixels.Bounds().Size() == nwsz {
		return
	}
	sp.Pixels = image.NewRGBA(image.Rectangle{Max: nwsz})
	if sp.Bg != nil && sp.Bg.Bounds().Size() != nwsz {
		sp.Bg = image.NewRGBA(image.Rectangle{Max: nwsz})
	}
}

// SetBottomPos sets the sprite's bottom position to given point
// the Geom.Pos represents its top position
func (sp *Sprite) SetBottomPos(pos image.Point) {
	sp.Geom.Pos = pos
	sp.Geom.Pos.Y -= sp.Geom.Size.Y
	sp.Geom.Pos.Y = ints.MaxInt(sp.Geom.Pos.Y, 0)
	sp.Geom.Pos.X = ints.MaxInt(sp.Geom.Pos.X, 0)
}

// GrabRenderFrom grabs the rendered image from given node
func (sp *Sprite) GrabRenderFrom(nii Node2D) {
	img := GrabRenderFrom(nii) // in bitmap.go
	if img != nil {
		sp.Pixels = img
		sp.Geom.Size = sp.Pixels.Bounds().Size()
	} else {
		sp.Resize(image.Point{10, 10}) // just a blank something..
	}
}
