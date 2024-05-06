// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"

	"cogentcore.org/core/base/ordmap"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/system"
	"cogentcore.org/core/vgpu/szalloc"
	"golang.org/x/image/draw"
)

// A Sprite is just an image (with optional background) that can be drawn onto
// the OverTex overlay texture of a window.  Sprites are used for text cursors/carets
// and for dynamic editing / interactive GUI elements (e.g., drag-n-drop elements)
type Sprite struct {

	// whether this sprite is active now or not
	Active bool

	// unique name of sprite
	Name string

	// properties for sprite, which allow for user-extensible data
	Properties map[string]any

	// position and size of the image within the RenderWindow
	Geom math32.Geom2DInt

	// pixels to render -- should be same size as Geom.Size
	Pixels *image.RGBA

	// Listeners are event listener functions for processing events on this widget.
	// They are called in sequential descending order (so the last added listener
	// is called first). They should be added using the On function. FirstListeners
	// and FinalListeners are called before and after these listeners, respectively.
	Listeners events.Listeners `copier:"-" json:"-" xml:"-" set:"-"`
}

// NewSprite returns a new sprite with given name, which must remain
// invariant and unique among all sprites in use, and is used for all access;
// prefix with package and type name to ensure uniqueness.  Starts out in
// inactive state; must call ActivateSprite.  If size is 0, no image is made.
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

// GrabRenderFrom grabs the rendered image from the given widget.
func (sp *Sprite) GrabRenderFrom(w Widget) {
	img := GrabRenderFrom(w)
	if img != nil {
		sp.Pixels = img
		sp.Geom.Size = sp.Pixels.Bounds().Size()
	} else {
		sp.SetSize(image.Pt(10, 10)) // just a blank placeholder
	}
}

// GrabRenderFrom grabs the rendered image from the given widget.
// If it returns nil, then the image could not be fetched.
func GrabRenderFrom(w Widget) *image.RGBA {
	wb := w.AsWidget()
	if wb.Scene.Pixels == nil {
		return nil
	}
	if wb.Geom.TotalBBox.Empty() { // the widget is offscreen
		return nil
	}
	sz := wb.Geom.TotalBBox.Size()
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), wb.Scene.Pixels, wb.Geom.TotalBBox.Min, draw.Src)
	return img
}

// On adds the given event handler to the sprite's Listeners for the given event type.
// Listeners are called in sequential descending order, so this listener will be called
// before all of the ones added before it.
func (sp *Sprite) On(etype events.Types, fun func(e events.Event)) *Sprite {
	sp.Listeners.Add(etype, fun)
	return sp
}

// OnClick adds an event listener function for [events.Click] events
func (sp *Sprite) OnClick(fun func(e events.Event)) *Sprite {
	return sp.On(events.Click, fun)
}

// OnSlideMove adds an event listener function for [events.SlideMove] events
func (sp *Sprite) OnSlideMove(fun func(e events.Event)) *Sprite {
	return sp.On(events.SlideMove, fun)
}

// OnSlideStop adds an event listener function for [events.SlideStop] events
func (sp *Sprite) OnSlideStop(fun func(e events.Event)) *Sprite {
	return sp.On(events.SlideStop, fun)
}

// Send sends an NEW event of given type to this sprite,
// optionally starting from values in the given original event
// (recommended to include where possible).
// Do NOT send an existing event using this method if you
// want the Handled state to persist throughout the call chain;
// call HandleEvent directly for any existing events.
func (sp *Sprite) HandleEvent(e events.Event) {
	sp.Listeners.Call(e)
}

// Send sends an NEW event of given type to this sprite,
// optionally starting from values in the given original event
// (recommended to include where possible).
// Do NOT send an existing event using this method if you
// want the Handled state to persist throughout the call chain;
// call HandleEvent directly for any existing events.
func (sp *Sprite) Send(typ events.Types, orig ...events.Event) {
	var e events.Event
	if len(orig) > 0 && orig[0] != nil {
		e = orig[0].NewFromClone(typ)
	} else {
		e = &events.Base{Typ: typ}
		e.Init()
	}
	sp.HandleEvent(e)
}

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
		sp := kv.Value
		if sp.Geom.Size == (image.Point{}) {
			continue
		}
		szs[idx] = sp.Geom.Size
		idx++
	}
	if idx != ns {
		szs = szs[:idx]
	}
	ss.SzAlloc.SetSizes(image.Point{4, 4}, system.MaxImageLayers, szs)
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
		sp := kv.Value
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
	return ss.Names.ValueByKeyTry(name)
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
	if !sp.Active {
		sp.Active = true
		ss.Modified = true
	}
}

// InactivateSprite flags the sprite as inactive, setting Modified if wasn't before
func (ss *Sprites) InactivateSprite(name string) {
	sp, ok := ss.SpriteByName(name)
	if !ok {
		return // not worth bothering about errs -- use a consistent string var!
	}
	if sp.Active {
		sp.Active = false
		ss.Modified = true
	}
}

// InactivateAllSprites inactivates all sprites, setting Modified if wasn't before
func (ss *Sprites) InactivateAllSprites() {
	for _, sp := range ss.Names.Order {
		if sp.Value.Active {
			sp.Value.Active = false
			ss.Modified = true
		}
	}
}

// ConfigSprites updates the Drawer configuration of sprites.
// Does a new SzAlloc, and sets corresponding images.
func (ss *Sprites) ConfigSprites(drw system.Drawer) {
	// fmt.Println("config sprites")
	ss.AllocSizes()
	sa := &ss.SzAlloc
	for gpi, ga := range sa.GpAllocs {
		gsz := sa.GpSizes[gpi]
		imgidx := SpriteStart + gpi
		drw.ConfigImageDefaultFormat(imgidx, gsz.X, gsz.Y, len(ga))
		for ii, spi := range ga {
			if err := ss.Names.IndexIsValid(spi); err != nil {
				fmt.Println(err)
				continue
			}
			sp := ss.Names.ValueByIndex(spi)
			drw.SetGoImage(imgidx, ii, sp.Pixels, system.NoFlipY)
		}
	}
	ss.Modified = false
}

// DrawSprites draws sprites
func (ss *Sprites) DrawSprites(drw system.Drawer) {
	// fmt.Println("draw sprites")
	sa := &ss.SzAlloc
	for gpi, ga := range sa.GpAllocs {
		imgidx := SpriteStart + gpi
		for ii, spi := range ga {
			if ss.Names.IndexIsValid(spi) != nil {
				continue
			}
			sp := ss.Names.ValueByIndex(spi)
			if !sp.Active {
				continue
			}
			// fmt.Println("ds", imgidx, ii, sp.Geom.Pos)
			drw.Copy(imgidx, ii, sp.Geom.Pos, image.Rectangle{}, draw.Over, system.NoFlipY)
		}
	}
}
