// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"

	"cogentcore.org/core/base/ordmap"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/system"
	"golang.org/x/image/draw"
)

// A Sprite is just an image (with optional background) that can be drawn onto
// the OverTex overlay texture of a window.  Sprites are used for text cursors/carets
// and for dynamic editing / interactive GUI elements (e.g., drag-n-drop elements)
type Sprite struct {

	// Active is whether this sprite is Active now or not.
	Active bool

	// Name is the unique name of the sprite.
	Name string

	// properties for sprite, which allow for user-extensible data
	Properties map[string]any

	// position and size of the image within the RenderWindow
	Geom math32.Geom2DInt

	// pixels to render, which should be the same size as [Sprite.Geom.Size]
	Pixels *image.RGBA

	// listeners are event listener functions for processing events on this widget.
	// They are called in sequential descending order (so the last added listener
	// is called first). They should be added using the On function. FirstListeners
	// and FinalListeners are called before and after these listeners, respectively.
	listeners events.Listeners `copier:"-" json:"-" xml:"-" set:"-"`
}

// NewSprite returns a new [Sprite] with the given name, which must remain
// invariant and unique among all sprites in use, and is used for all access;
// prefix with package and type name to ensure uniqueness. Starts out in
// inactive state; must call ActivateSprite. If size is 0, no image is made.
func NewSprite(name string, sz image.Point, pos image.Point) *Sprite {
	sp := &Sprite{Name: name}
	sp.SetSize(sz)
	sp.Geom.Pos = pos
	return sp
}

// SetSize sets sprite image to given size; makes a new image (does not resize)
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

// grabRenderFrom grabs the rendered image from the given widget.
func (sp *Sprite) grabRenderFrom(w Widget) {
	img := grabRenderFrom(w)
	if img != nil {
		sp.Pixels = img
		sp.Geom.Size = sp.Pixels.Bounds().Size()
	} else {
		sp.SetSize(image.Pt(10, 10)) // just a blank placeholder
	}
}

// grabRenderFrom grabs the rendered image from the given widget.
// If it returns nil, then the image could not be fetched.
func grabRenderFrom(w Widget) *image.RGBA {
	wb := w.AsWidget()
	scimg := wb.Scene.Painter.RenderImage()
	if scimg == nil {
		return nil
	}
	if wb.Geom.TotalBBox.Empty() { // the widget is offscreen
		return nil
	}
	sz := wb.Geom.TotalBBox.Size()
	img := image.NewRGBA(image.Rectangle{Max: sz})
	draw.Draw(img, img.Bounds(), scimg, wb.Geom.TotalBBox.Min, draw.Src)
	return img
}

// On adds the given event handler to the sprite's Listeners for the given event type.
// Listeners are called in sequential descending order, so this listener will be called
// before all of the ones added before it.
func (sp *Sprite) On(etype events.Types, fun func(e events.Event)) *Sprite {
	sp.listeners.Add(etype, fun)
	return sp
}

// OnClick adds an event listener function for [events.Click] events
func (sp *Sprite) OnClick(fun func(e events.Event)) *Sprite {
	return sp.On(events.Click, fun)
}

// OnSlideStart adds an event listener function for [events.SlideStart] events
func (sp *Sprite) OnSlideStart(fun func(e events.Event)) *Sprite {
	return sp.On(events.SlideStart, fun)
}

// OnSlideMove adds an event listener function for [events.SlideMove] events
func (sp *Sprite) OnSlideMove(fun func(e events.Event)) *Sprite {
	return sp.On(events.SlideMove, fun)
}

// OnSlideStop adds an event listener function for [events.SlideStop] events
func (sp *Sprite) OnSlideStop(fun func(e events.Event)) *Sprite {
	return sp.On(events.SlideStop, fun)
}

// HandleEvent sends the given event to all listeners for that event type.
func (sp *Sprite) handleEvent(e events.Event) {
	sp.listeners.Call(e)
}

// send sends an new event of the given type to this sprite,
// optionally starting from values in the given original event
// (recommended to include where possible).
// Do not send an existing event using this method if you
// want the Handled state to persist throughout the call chain;
// call [Sprite.handleEvent] directly for any existing events.
func (sp *Sprite) send(typ events.Types, original ...events.Event) {
	var e events.Event
	if len(original) > 0 && original[0] != nil {
		e = original[0].NewFromClone(typ)
	} else {
		e = &events.Base{Typ: typ}
		e.Init()
	}
	sp.handleEvent(e)
}

// Sprites manages a collection of Sprites, with unique name ids.
type Sprites struct {
	ordmap.Map[string, *Sprite]

	// set to true if sprites have been modified since last config
	Modified bool
}

// Add adds sprite to list, and returns the image index and
// layer index within that for given sprite.  If name already
// exists on list, then it is returned, with size allocation
// updated as needed.
func (ss *Sprites) Add(sp *Sprite) {
	ss.Init()
	ss.Map.Add(sp.Name, sp)
	ss.Modified = true
}

// Delete deletes sprite by name, returning indexes where it was located.
// All sprite images must be updated when this occurs, as indexes may have shifted.
func (ss *Sprites) Delete(sp *Sprite) {
	ss.DeleteKey(sp.Name)
	ss.Modified = true
}

// SpriteByName returns the sprite by name
func (ss *Sprites) SpriteByName(name string) (*Sprite, bool) {
	return ss.ValueByKeyTry(name)
}

// reset removes all sprites
func (ss *Sprites) reset() {
	ss.Reset()
	ss.Modified = true
}

// ActivateSprite flags the sprite as active, setting Modified if wasn't before.
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

// InactivateSprite flags the sprite as inactive, setting Modified if wasn't before.
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

// renderSprites adds active sprites to given renderScene
func (ss *Sprites) renderSprites(renderScene *render.Scene, scpos image.Point) {
	for _, kv := range ss.Order {
		sp := kv.Value
		if !sp.Active {
			continue
		}
		// note: may need to copy pixels but hoping not..
		sr := &SpriteRender{DrawPos: sp.Geom.Pos.Add(scpos), Pixels: sp.Pixels}
		renderScene.Add(sr)
	}
	ss.Modified = false
}

// SpriteRender is a [render.Render] implementation for sprites.
type SpriteRender struct {
	DrawPos image.Point
	Pixels  *image.RGBA
}

func (sr *SpriteRender) Render() {}

func (sr *SpriteRender) Draw(drw system.Drawer) {
	drw.Copy(sr.DrawPos, sr.Pixels, sr.Pixels.Bounds(), draw.Over, system.Unchanged)
}
