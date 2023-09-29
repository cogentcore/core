// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/girl/gist"
	"goki.dev/goosi"
	"goki.dev/ki/v2"
	"goki.dev/ordmap"
	"goki.dev/vgpu/v2/szalloc"
	"goki.dev/vgpu/v2/vgpu"
)

// A Sprite is just an image (with optional background) that can be drawn onto
// the OverTex overlay texture of a window.  Sprites are used for cursors
// and for dynamic editing / interactive GUI elements (e.g., drag-n-drop elments)
type Sprite struct {

	// whether this sprite is active now or not
	On bool `desc:"whether this sprite is active now or not"`

	// unique name of sprite
	Name string `desc:"unique name of sprite"`

	// properties for sprite -- allows user-extensible data
	Props ki.Props `desc:"properties for sprite -- allows user-extensible data"`

	// position and size of the image within the overlay window texture
	Geom gist.Geom2DInt `desc:"position and size of the image within the overlay window texture"`

	// pixels to render -- should be same size as Geom.Size
	Pixels *image.RGBA `desc:"pixels to render -- should be same size as Geom.Size"`

	// optional event signals for given event type
	// Events map[goosi.EventTypes]*ki.Signal `desc:"optional event signals for given event type"`
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
	sp.Geom.Pos.Y = max(sp.Geom.Pos.Y, 0)
	sp.Geom.Pos.X = max(sp.Geom.Pos.X, 0)
}

// GrabRenderFrom grabs the rendered image from given node
func (sp *Sprite) GrabRenderFrom(wi Widget) {
	img := GrabRenderFrom(wi) // in bitmap.go
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
func (sp *Sprite) ConnectEvent(recv ki.Ki, et goosi.EventTypes, fun func()) {
	if sp.Events == nil {
		sp.Events = make(map[goosi.EventTypes]*ki.Signal)
	}
	sg, ok := sp.Events[et]
	if !ok {
		sg = &ki.Signal{}
		sp.Events[et] = sg
	}
	sg.Connect(recv, fun)
}

// DisconnectEvent removes Signal connection for given event type to given receiver.
func (sp *Sprite) DisconnectEvent(recv ki.Ki, et goosi.EventTypes, fun func()) {
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

	// map of uniquely named sprites
	Names ordmap.Map[string, *Sprite] `desc:"map of uniquely named sprites"`

	// allocation of sprites by size for rendering
	SzAlloc szalloc.SzAlloc `desc:"allocation of sprites by size for rendering"`

	// set to true if sprites have been modified since last config
	Modified bool `desc:"set to true if sprites have been modified since last config"`

	// number of active sprites
	Active int `desc:"number of active sprites"`
}

func (ss *Sprites) Init() {
	ss.Names.Init()
}

// AllocSizes allocates the sprites by size to fixed set of images and layers
func (ss *Sprites) AllocSizes() {
	ns := ss.Names.Len()
	szs := make([]image.Point, ns)
	idx := 0
	for _, kv := range ss.Names.Order {
		sp := kv.Val
		if sp.Geom.Size == (image.Point{}) {
			continue
		}
		szs[idx] = sp.Geom.Size
		idx++
	}
	if idx != ns {
		szs = szs[:idx]
	}
	ss.SzAlloc.SetSizes(image.Point{4, 4}, vgpu.MaxImageLayers, szs)
	ss.SzAlloc.Alloc()
}

// Add adds sprite to list, and returns the image index and
// layer index within that for given sprite.  If name already
// exists on list, then it is returned, with size allocation
// updated as needed.
func (ss *Sprites) Add(sp *Sprite) {
	ss.Init()
	ss.Names.Add(sp.Name, sp)
	ss.Modified = true
}

// Delete deletes sprite by name, returning indexes where it was located.
// All sprite images must be updated when this occurs, as indexes may have shifted.
func (ss *Sprites) Delete(sp *Sprite) {
	ss.Names.DeleteKey(sp.Name)
	ss.Modified = true
	return
}

// HasSizeChanged returns true if a sprite's size has changed relative to
// its last allocated value, in SzAlloc.  Returns true and sets Modified
// flag to true if so.
func (ss *Sprites) HasSizeChanged() bool {
	if len(ss.SzAlloc.ItemSizes) != ss.Names.Len() {
		ss.Modified = true
		return true
	}
	for i, kv := range ss.Names.Order {
		sp := kv.Val
		ssz := ss.SzAlloc.ItemSizes[i]
		if sp.Geom.Size != ssz {
			ss.Modified = true
			return true
		}
	}
	return false
}

// SpriteByName returns the sprite by name
func (ss *Sprites) SpriteByName(name string) (*Sprite, bool) {
	return ss.Names.ValByKeyTry(name)
}

// Reset removes all sprites
func (ss *Sprites) Reset() {
	ss.Names.Reset()
	ss.Modified = true
}
