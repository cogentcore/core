// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/kigen/ordmap"
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

// NewSprite returns a new sprite with given name, which must remain
// invariant and unique among all sprites in use, and is used for all access
// -- prefix with package and type name to ensure uniqueness.  Starts out in
// inactive state -- must call ActivateSprite.  If size is 0, no image is made.
func NewSprite(nm string, sz image.Point, pos image.Point) *Sprite {
	sp := &Sprite{Name: nm}
	sp.SetSize(sz)
	sp.Geom.Pos = pos
	return sp
}

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

/////////////////////////////////////////////////////////////////////
// Sprites

// Sprites manages a collection of sprites organized by size and name
type Sprites struct {
	Names  ordmap.Map[string, *Sprite]                           `desc:"map of uniquely named sprites"`
	Sizes  ordmap.Map[image.Point, *ordmap.Map[string, *Sprite]] `desc:"sprite list organized by size -- index gives Texture to use"`
	Active int                                                   `desc:"number of active sprites"`
}

func (ss *Sprites) Init() {
	ss.Names.Init()
	ss.Sizes.Init()
}

// NSizes returns the number of current sizes allocated
func (ss *Sprites) NSizes() int {
	return ss.Sizes.Len()
}

// todo: update image method

// Add adds sprite to list, and returns the image index and
// layer index within that for given sprite.  If name already
// exists on list, then it is returned, with size allocation
// updated as needed.
func (ss *Sprites) Add(sp *Sprite) (imgIdx, layIdx int) {
	ss.Init()
	esi, has := ss.Names.IdxByKey(sp.Name)
	if has {
		esp := ss.Names.Order[esi].Val
		ss.Names.ReplaceIdx(esi, sp.Name, sp)
		if esp.Geom.Size == sp.Geom.Size {
			return ss.AddSize(sp) // auto replaces
		}
		ss.DeleteSize(esp)
		return ss.AddSize(sp)
	}
	ss.Names.Add(sp.Name, sp)
	return ss.AddSize(sp)
}

// Delete deletes sprite by name, returning indexes where it was located.
// All sprite images must be updated when this occurs, as indexes may have shifted.
func (ss *Sprites) Delete(sp *Sprite) (imgIdx, layIdx int) {
	imgIdx, layIdx = ss.Indexes(sp)
	if imgIdx < 0 {
		return
	}
	ss.DeleteSize(sp)
	ss.Names.DeleteKey(sp.Name)
	return
}

// AddSize adds sprite to list of sprites organized by size
func (ss *Sprites) AddSize(sp *Sprite) (imgIdx, layIdx int) {
	has := false
	sz := sp.Geom.Size
	var sm *ordmap.Map[string, *Sprite]
	imgIdx, has = ss.Sizes.IdxByKey(sz)
	if !has {
		sm = &ordmap.Map[string, *Sprite]{}
		sm.Init()
		imgIdx = ss.Sizes.Len()
		ss.Sizes.Add(sz, sm)
	} else {
		sm = ss.Sizes.Order[imgIdx].Val
	}
	layIdx, has = sm.IdxByKey(sp.Name)
	if !has {
		layIdx = sm.Len()
		sm.Add(sp.Name, sp)
	} else {
		sm.ReplaceIdx(layIdx, sp.Name, sp)
	}
	return
}

// DeleteSize removes sprite from Sizes list based on its sizser
func (ss *Sprites) DeleteSize(sp *Sprite) {
	sz := sp.Geom.Size
	sm, has := ss.Sizes.ValByKey(sz)
	if !has { // error!?
		return
	}
	sm.DeleteKey(sp.Name)
}

// SpriteByName returns the sprite by name
func (ss *Sprites) SpriteByName(name string) (*Sprite, bool) {
	return ss.Names.ValByKey(name)
}

// Indexes returns the texture image index (size list)
// and layer index (name list within size) for given sprite
// returns -1 if not found (assumed to only be called when exists)
func (ss *Sprites) Indexes(sp *Sprite) (imgIdx, layIdx int) {
	sz := sp.Geom.Size
	has := false
	imgIdx, has = ss.Sizes.IdxByKey(sz)
	if !has { // error!?
		return -1, -1
	}
	sm := ss.Sizes.Order[imgIdx].Val
	layIdx, has = sm.IdxByKey(sp.Name)
	if !has { // error!?
		return -1, -1
	}
	return
}

// Reset removes all sprites
func (ss *Sprites) Reset() {
	ss.Names.Reset()
	ss.Sizes.Reset()
}
