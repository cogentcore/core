// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"goki.dev/goosi"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/ordmap"
	"goki.dev/vgpu/v2/szalloc"
	"golang.org/x/image/draw"
)

// A Sprite is just an image (with optional background) that can be drawn onto
// the OverTex overlay texture of a window.  Sprites are used for cursors
// and for dynamic editing / interactive GUI elements (e.g., drag-n-drop elments)
type Sprite struct {

	// whether this sprite is active now or not
	On bool

	// unique name of sprite
	Name string

	// properties for sprite -- allows user-extensible data
	Props ki.Props

	// position and size of the image within the overlay window texture
	Geom mat32.Geom2DInt

	// pixels to render -- should be same size as Geom.Size
	Pixels *image.RGBA
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

// todo: replace with events

// ConnectEvent adds a Signal connection for given event type to given receiver.
// only mouse events are supported.
// Sprite events are always top priority -- if mouse is inside sprite geom, then it is sent
// if event function does not mark event as processed, it will continue to propagate
// func (sp *Sprite) ConnectEvent(recv ki.Ki, et events.Types, fun func()) {
// 	if sp.Events == nil {
// 		sp.Events = make(map[events.Types]*ki.Signal)
// 	}
// 	sg, ok := sp.Events[et]
// 	if !ok {
// 		sg = &ki.Signal{}
// 		sp.Events[et] = sg
// 	}
// 	sg.Connect(recv, fun)
// }
//
// // DisconnectEvent removes Signal connection for given event type to given receiver.
// func (sp *Sprite) DisconnectEvent(recv ki.Ki, et events.Types, fun func()) {
// 	if sp.Events == nil {
// 		return
// 	}
// 	sg, ok := sp.Events[et]
// 	if !ok {
// 		return
// 	}
// 	sg.Disconnect(recv)
// }
//
// // DisconnectAllEvents removes all event connections for this sprite
// func (sp *Sprite) DisconnectAllEvents() {
// 	sp.Events = nil
// }

/////////////////////////////////////////////////////////////////////
// Sprites

// Sprites manages a collection of sprites organized by size and name
type Sprites struct {

	// map of uniquely named sprites
	Names ordmap.Map[string, *Sprite]

	// allocation of sprites by size for rendering
	SzAlloc szalloc.SzAlloc

	// set to true if sprites have been modified since last config
	Modified bool

	// number of active sprites
	Active int
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
	ss.SzAlloc.SetSizes(image.Point{4, 4}, goosi.MaxImageLayers, szs)
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

// ActivateSprite flags the sprite as active, setting Modified if wasn't before
func (ss *Sprites) ActivateSprite(name string) {
	sp, ok := ss.SpriteByName(name)
	if !ok {
		return // not worth bothering about errs -- use a consistent string var!
	}
	if !sp.On {
		sp.On = true
		ss.Modified = true
	}
}

// InactivateSprite flags the sprite as inactive, setting Modified if wasn't before
func (ss *Sprites) InactivateSprite(name string) {
	sp, ok := ss.SpriteByName(name)
	if !ok {
		return // not worth bothering about errs -- use a consistent string var!
	}
	if sp.On {
		sp.On = false
		ss.Modified = true
	}
}

// InactivateAllSprites inactivates all sprites, setting Modified if wasn't before
func (ss *Sprites) InactivateAllSprites() {
	for _, sp := range ss.Names.Order {
		if sp.Val.On {
			sp.Val.On = false
			ss.Modified = true
		}
	}
}

// ConfigSprites updates the Drawer configuration of sprites.
// Does a new SzAlloc, and sets corresponding images.
func (ss *Sprites) ConfigSprites(drw goosi.Drawer) {
	// fmt.Println("config sprites")
	ss.AllocSizes()
	sa := &ss.SzAlloc
	for gpi, ga := range sa.GpAllocs {
		gsz := sa.GpSizes[gpi]
		imgidx := SpriteStart + gpi
		drw.ConfigImageDefaultFormat(imgidx, gsz.X, gsz.Y, len(ga))
		for ii, spi := range ga {
			if err := ss.Names.IdxIsValid(spi); err != nil {
				fmt.Println(err)
				continue
			}
			sp := ss.Names.ValByIdx(spi)
			drw.SetGoImage(imgidx, ii, sp.Pixels, goosi.NoFlipY)
		}
	}
	ss.Modified = false
}

// DrawSprites draws sprites
func (ss *Sprites) DrawSprites(drw goosi.Drawer) {
	// fmt.Println("draw sprites")
	sa := &ss.SzAlloc
	for gpi, ga := range sa.GpAllocs {
		imgidx := SpriteStart + gpi
		for ii, spi := range ga {
			if ss.Names.IdxIsValid(spi) != nil {
				continue
			}
			sp := ss.Names.ValByIdx(spi)
			if !sp.On {
				continue
			}
			// fmt.Println("ds", imgidx, ii, sp.Geom.Pos)
			drw.Copy(imgidx, ii, sp.Geom.Pos, image.Rectangle{}, draw.Over, goosi.NoFlipY)
		}
	}
}

// SpriteEvent processes given event for any active sprites
// func (ss *Sprites) SelSpriteEvent(evi events.Event) {
// 	// w.StageMgr.RenderCtx.Mu.Lock()
// 	// defer w.StageMgr.RenderCtx.Mu.Unlock()
//
// 	et := evi.Type()
//
// 	for _, spkv := range w.Sprites.Names.Order {
// 		sp := spkv.Val
// 		if !sp.On {
// 			continue
// 		}
// 		if sp.Events == nil {
// 			continue
// 		}
// 		sig, ok := sp.Events[et]
// 		if !ok {
// 			continue
// 		}
// 		ep := evi.Pos()
// 		if et == events.EventsDragEvent {
// 			if sp.Name == w.SpriteDragging {
// 				sig.Emit(w.This(), int64(et), evi)
// 			}
// 		} else if ep.In(sp.Geom.Bounds()) {
// 			sig.Emit(w.This(), int64(et), evi)
// 		}
// 	}
// }
