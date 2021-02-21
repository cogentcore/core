// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
)

// A Sprite is just an image (with optional background) that can be drawn onto
// the OverTex overlay texture of a window.  Sprites are used for cursors
// and for dynamic editing / interactive GUI elements (e.g., drag-n-drop elments)
type Sprite struct {
	On     bool                           `desc:"whether this sprite is active now or not"`
	Name   string                         `desc:"unique name of sprite"`
	Props  ki.Props                       `desc:"properties for sprite -- allows user-extensible data"`
	Geom   Geom2DInt                      `desc:"position and size of the image within the overlay window texture"`
	Pixels *image.RGBA                    `desc:"pixels to render -- should be same size as Geom.Size"`
	Events map[oswin.EventType]*ki.Signal `desc:"optional event signals for given event type"`
}

// Sprites is a map of named Sprite elements
type Sprites map[string]*Sprite

// SetSize sets sprite image to given size -- makes a new image (does not resize)
// returns true if a new image was set
func (sp *Sprite) SetSize(nwsz image.Point) bool {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return false
	}
	sp.Geom.Size = nwsz // always make sure
	if sp.Pixels != nil && sp.Pixels.Bounds().Size() == nwsz {
		return false
	}
	sp.Pixels = image.NewRGBA(image.Rectangle{Max: nwsz})
	return true
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
		sp.SetSize(image.Point{10, 10}) // just a blank something..
	}
}

// ConnectEvent adds a Signal connection for given event type to given receiver.
// only mouse events are supported.
// Sprite events are always top priority -- if mouse is inside sprite geom, then it is sent
// if event function does not mark event as processed, it will continue to propagate
func (sp *Sprite) ConnectEvent(recv ki.Ki, et oswin.EventType, fun ki.RecvFunc) {
	if sp.Events == nil {
		sp.Events = make(map[oswin.EventType]*ki.Signal)
	}
	sg, ok := sp.Events[et]
	if !ok {
		sg = &ki.Signal{}
		sp.Events[et] = sg
	}
	sg.Connect(recv, fun)
}

// DisconnectEvent removes Signal connection for given event type to given receiver.
func (sp *Sprite) DisconnectEvent(recv ki.Ki, et oswin.EventType, fun ki.RecvFunc) {
	if sp.Events == nil {
		return
	}
	sg, ok := sp.Events[et]
	if !ok {
		return
	}
	sg.Disconnect(recv)
}

// DisconnectAllEvents removes all event connections for this sprite
func (sp *Sprite) DisconnectAllEvents() {
	sp.Events = nil
}
