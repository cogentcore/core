// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"sync"

	"cogentcore.org/core/base/keylist"
	"cogentcore.org/core/base/tiered"
	"cogentcore.org/core/events"
	"cogentcore.org/core/paint"
	"golang.org/x/image/draw"
)

// A Sprite is a top-level rendering element that paints onto a transparent
// layer that is cleared every render pass. Sprites are used for text cursors/carets
// and for dynamic editing / interactive GUI elements (e.g., drag-n-drop elements).
// To support cursor sprites and other animations, the sprites are redrawn at a
// minimum update rate that is at least as fast as CursorBlinkTime.
// Sprites can also receive mouse events, within their event bounding box.
// It is basically like a [Canvas] element over the entire screen, with no constraints
// on where you can draw.
type Sprite struct {

	// Active is whether this sprite is Active now or not.
	// Active sprites Draw and can receive events.
	Active bool

	// Name is the unique name of the sprite.
	Name string

	// properties for sprite, which allow for user-extensible data
	Properties map[string]any

	// Draw is the function that is called for Active sprites on every render pass
	// to draw the sprite onto the top-level transparent layer.
	Draw func(pc *paint.Painter)

	// EventBBox is the bounding box for this sprite to receive mouse events.
	// Typically this is the region in which it renders.
	EventBBox image.Rectangle

	// listeners are event listener functions for processing events on this widget.
	// They are called in sequential descending order (so the last added listener
	// is called first). They should be added using the On function. FirstListeners
	// and FinalListeners are called before and after these listeners, respectively.
	listeners events.Listeners `copier:"-" json:"-" xml:"-" set:"-"`
}

// NewSprite returns a new [Sprite] with the given name, which must remain
// invariant and unique among all sprites in use, and is used for all access;
// prefix with package and type name to ensure uniqueness. Starts out in
// inactive state; must call ActivateSprite.
func NewSprite(name string, draw func(pc *paint.Painter)) *Sprite {
	sp := &Sprite{Name: name, Draw: draw}
	return sp
}

// InitProperties ensures that the Properties map exists.
func (sp *Sprite) InitProperties() {
	if sp.Properties != nil {
		return
	}
	sp.Properties = make(map[string]any)
}

// SetPos sets the position of the sprite EventBBox, keeping the same size.
func (sp *Sprite) SetPos(pos image.Point) {
	sp.EventBBox = sp.EventBBox.Add(pos.Sub(sp.EventBBox.Min))
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

// NewImageSprite returns a new Sprite that draws the given image
// in the given location, which is stored in the Min of the EventBBox.
// Move the EventBBox to move the render location.
// The image is stored as "image" in Properties.
func NewImageSprite(name string, pos image.Point, img image.Image) *Sprite {
	sp := &Sprite{Name: name}
	sp.InitProperties()
	sp.Properties["image"] = img
	sp.EventBBox = img.Bounds().Add(pos)
	sp.Draw = func(pc *paint.Painter) {
		pc.DrawImage(img, sp.EventBBox, image.Point{}, draw.Over)
	}
	return sp
}

////////  Sprites

type SpriteList = keylist.List[string, *Sprite]

// Sprites manages a collection of Sprites, with unique name ids within each
// of three priority lists: Normal, First and Final. The convenience API operates
// on the Normal list, while First and Final are available for more advanced cases
// where rendering order needs to be controlled. First items are rendered first and
// processed first for event handling, and likewise for Final.
type Sprites struct {
	tiered.Tiered[SpriteList]

	// set to true if sprites have been modified since last config
	modified bool

	sync.Mutex
}

// SetModified sets the sprite modified flag, which will
// drive a render to reflect the updated sprite.
// This version locks the sprites: see also [Sprites.SetModifiedLocked].
func (ss *Sprites) SetModified() {
	ss.Lock()
	ss.modified = true
	ss.Unlock()
}

// SetModifiedLocked sets the sprite modified flag, which will
// drive a render to reflect the updated sprite.
// This version assumes Sprites are already locked, which is better for
// doing multiple coordinated updates at the same time.
func (ss *Sprites) SetModifiedLocked() {
	ss.modified = true
}

// IsModified returns whether the sprites have been modified, under lock.
func (ss *Sprites) IsModified() bool {
	ss.Lock()
	defer ss.Unlock()
	return ss.modified
}

// Add adds sprite to the Normal list of sprites, updating if already there.
// This version locks the sprites: see also [Sprites.AddLocked].
func (ss *Sprites) Add(sp *Sprite) {
	ss.Lock()
	ss.AddLocked(sp)
	ss.Unlock()
}

// AddLocked adds sprite to the Normal list of sprites, updating if already there.
// This version assumes Sprites are already locked, which is better for
// doing multiple coordinated updates at the same time.
func (ss *Sprites) AddLocked(sp *Sprite) {
	ss.Normal.Add(sp.Name, sp)
	ss.modified = true
}

// Delete deletes given sprite by name from the Normal list.
// This version locks the sprites: see also [Sprites.DeleteLocked].
func (ss *Sprites) Delete(name string) {
	ss.Lock()
	ss.DeleteLocked(name)
	ss.Unlock()
}

// DeleteLocked deletes given sprite by name from the Normal list.
// This version assumes Sprites are already locked, which is better for
// doing multiple coordinated updates at the same time.
func (ss *Sprites) DeleteLocked(name string) {
	ss.Normal.DeleteByKey(name)
	ss.modified = true
}

// SpriteByName returns the Normal sprite by name.
// This version locks the sprites: see also [Sprites.SpriteByNameLocked].
func (ss *Sprites) SpriteByName(name string) (*Sprite, bool) {
	ss.Lock()
	defer ss.Unlock()
	return ss.SpriteByNameLocked(name)
}

// SpriteByNameLocked returns the Normal sprite by name.
// This version assumes Sprites are already locked, which is better for
// doing multiple coordinated updates at the same time.
func (ss *Sprites) SpriteByNameLocked(name string) (*Sprite, bool) {
	return ss.Normal.AtTry(name)
}

// reset removes all sprites.
func (ss *Sprites) reset() {
	ss.Lock()
	ss.Do(func(sl SpriteList) {
		sl.Reset()
	})
	ss.modified = true
	ss.Unlock()
}

// ActivateSprite flags the Normal sprite(s) as active, setting Modified if wasn't before.
// This version locks the sprites: see also [Sprites.ActivateSpriteLocked].
func (ss *Sprites) ActivateSprite(name ...string) {
	ss.Lock()
	ss.ActivateSpriteLocked(name...)
	ss.Unlock()
}

// ActivateSpriteLocked flags the Normal sprite(s) as active,
// setting Modified if wasn't before.
// This version assumes Sprites are already locked, which is better for
// doing multiple coordinated updates at the same time.
func (ss *Sprites) ActivateSpriteLocked(name ...string) {
	for _, nm := range name {
		sp, ok := ss.SpriteByNameLocked(nm)
		if ok && !sp.Active {
			sp.Active = true
			ss.modified = true
		}
	}
}

// InactivateSprite flags the Normal sprite(s) as inactive,
// setting Modified if wasn't before.
// This version locks the sprites: see also [Sprites.InactivateSpriteLocked].
func (ss *Sprites) InactivateSprite(name ...string) {
	ss.Lock()
	ss.InactivateSpriteLocked(name...)
	ss.Unlock()
}

// InactivateSpriteLocked flags the Normal sprite(s) as inactive,
// setting Modified if wasn't before.
// This version assumes Sprites are already locked, which is better for
// doing multiple coordinated updates at the same time.
func (ss *Sprites) InactivateSpriteLocked(name ...string) {
	for _, nm := range name {
		sp, ok := ss.SpriteByNameLocked(nm)
		if ok && sp.Active {
			sp.Active = false
			ss.modified = true
		}
	}
}
