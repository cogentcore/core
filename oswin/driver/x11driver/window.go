// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x11driver

// TODO: implement a back buffer.

import (
	"image"
	"image/color"
	"image/draw"
	"sync"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/xproto"

	"github.com/goki/goki/gi/oswin"
	"github.com/goki/goki/gi/oswin/key"
	"github.com/goki/goki/gi/oswin/mouse"
	"github.com/goki/goki/gi/oswin/paint"
	"github.com/goki/goki/gi/oswin/window"
	"github.com/goki/goki/gi/oswin/driver/internal/drawer"
	"github.com/goki/goki/gi/oswin/driver/internal/event"
	"github.com/goki/goki/gi/oswin/driver/internal/lifecycler"
	"github.com/goki/goki/gi/oswin/driver/internal/x11key"
	"golang.org/x/image/math/f64"
)

type windowImpl struct {
	oswin.WindowBase

	app *appImpl

	// xw this is the id for windows
	xw xproto.Window
	xg xproto.Gcontext
	xp render.Picture

	event.Deque
	xevents chan xgb.Event

	// This next group of variables are mutable, but are only modified in the
	// appImpl.run goroutine.

	lifecycler lifecycler.State

	mu       sync.Mutex
	released bool
}

func (w *windowImpl) Release() {
	w.mu.Lock()
	released := w.released
	w.released = true
	w.mu.Unlock()

	// TODO: call w.lifecycler.SetDead and w.lifecycler.SendEvent, a la
	// handling atomWMDeleteWindow?

	if released {
		return
	}
	render.FreePicture(w.app.xc, w.xp)
	xproto.FreeGC(w.app.xc, w.xg)
	xproto.DestroyWindow(w.app.xc, w.xw)
}

func (w *windowImpl) Upload(dp image.Point, src oswin.Image, sr image.Rectangle) {
	src.(*imageImpl).upload(xproto.Drawable(w.xw), w.xg, w.app.xsi.RootDepth, dp, sr)
}

func (w *windowImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	fill(w.app.xc, w.xp, dr, src, op)
}

func (w *windowImpl) DrawUniform(src2dst f64.Aff3, src color.Color, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	w.app.drawUniform(w.xp, &src2dst, src, sr, op, opts)
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	src.(*textureImpl).draw(w.xp, &src2dst, sr, op, opts)
}

func (w *windowImpl) Copy(dp image.Point, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	drawer.Copy(w, dp, src, sr, op, opts)
}

func (w *windowImpl) Scale(dr image.Rectangle, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	drawer.Scale(w, dr, src, sr, op, opts)
}

func (w *windowImpl) Publish() oswin.PublishResult {
	// TODO: implement a back buffer, and copy or flip that here to the front
	// buffer.

	// This sync isn't needed to flush the outgoing X11 requests. Instead, it
	// acts as a form of flow control. Outgoing requests can be quite small on
	// the wire, e.g. draw this texture ID (an integer) to this rectangle (four
	// more integers), but much more expensive on the server (blending a
	// million source and destination pixels). Without this sync, the Go X11
	// client could easily end up sending work at a faster rate than the X11
	// server can serve.
	w.app.xc.Sync()

	return oswin.PublishResult{}
}

func (w *windowImpl) handleConfigureNotify(ev xproto.ConfigureNotifyEvent) {
	// TODO: does the order of these lifecycle and window events matter? Should
	// they really be a single, atomic event?
	w.lifecycler.SetVisible((int(ev.X)+int(ev.Width)) > 0 && (int(ev.Y)+int(ev.Height)) > 0)
	w.lifecycler.SendEvent(w, nil)

	// todo:
	// dpi := 25.4 * (float32(displayWidth) / float32(displayWidthMM))
	dpi := float32(96)
	ldpi := oswin.LogicalFmPhysicalDPI(dpi)

	sz := image.Point{int(ev.Width), int(ev.Height)}
	ps := image.Point{int(ev.X), int(ev.Y)}

	act := window.ActionN

	if w.Sz != sz || w.PhysDPI != dpi || w.LogDPI != ldpi {
		act = window.Resize
	} else if w.Pos != ps {
		act = window.Move
	} else {
		act = window.Resize // todo: for now safer to default to resize -- to catch the filtering
	}

	w.Sz = sz
	w.Pos = ps
	w.PhysDPI = dpi
	w.LogDPI = ldpi

	// if scrno > 0 && len(theApp.screens) > int(scrno) {
	// 	w.Scrn = theApp.screens[scrno]
	// }

	winEv := &window.Event{
		Size:       sz,
		LogicalDPI: ldpi,
		Action:     act,
	}
	winEv.Init()
	w.Send(winEv)
}

func (w *windowImpl) handleExpose() {
	w.Send(&paint.Event{})
}

func (w *windowImpl) handleKey(detail xproto.Keycode, state uint16, act key.Action) {
	r, c := w.app.keysyms.Lookup(uint8(detail), state)

	event := &key.Event{
		Rune:      r,
		Code:      c,
		Modifiers: x11key.KeyModifiers(state),
		Action:    act,
	}
	event.Init()
	w.Send(event)

	// do ChordEvent -- only for non-modifier key presses -- call
	// key.ChordString to convert the event into a parsable string for GUI
	// events
	if act == key.Press && !key.CodeIsModifier(c) {
		che := &key.ChordEvent{Event: *event}
		w.Send(che)
	}

}


var lastMouseClickEvent oswin.Event
var lastMouseEvent oswin.Event

func (w *windowImpl) handleMouse(x, y int16, button xproto.Button, state uint16, dir mouse.Action) {
	where := image.Point{int(x), int(y)}
	from := image.ZP
	if lastMouseEvent != nil {
		from = lastMouseEvent.Pos()
	}
	mods := x11key.KeyModifiers(state)
	stb := mouse.Button(x11key.ButtonFromState(state))

	var event oswin.Event
	switch {
	case button == 0: // moved
		if stb > 0 { // drag
			event = &mouse.DragEvent{
				MoveEvent: mouse.MoveEvent{
					Event: mouse.Event{
						Where:     where,
						Button:    stb,
						Action:    mouse.Drag,
						Modifiers: mods,
					},
					From: from,
				},
			}
		} else {
			event = &mouse.MoveEvent{
				Event: mouse.Event{
					Where:     where,
					Button:    mouse.NoButton,
					Action:    mouse.Move,
					Modifiers: mods,
				},
				From: from,
			}
		}
	case button < 4: // regular click
		act := mouse.Action(dir)
		if act == mouse.Press && lastMouseClickEvent != nil {
			interval := time.Now().Sub(lastMouseClickEvent.Time())
			// fmt.Printf("interval: %v\n", interval)
			if (interval / time.Millisecond) < time.Duration(mouse.DoubleClickMSec) {
				act = mouse.DoubleClick
			}
		}
		event = &mouse.Event{
			Where:     where,
			Button:    mouse.Button(button),
			Action:    act,
			Modifiers: mods,
		}
		if act == mouse.Press {
			event.SetTime()
			lastMouseClickEvent = event
		}
	default: // scroll wheel, 4-7
		if dir != mouse.Press { // only care about these for scrolling
			return
		}
		del := image.Point{}
		switch button {
		case 4: // up
			del.Y = -mouse.ScrollWheelRate
		case 5: // down
			del.Y = mouse.ScrollWheelRate
		case 6: // left
			del.X = -mouse.ScrollWheelRate
		case 7: // right
			del.X = mouse.ScrollWheelRate
		}
		event = &mouse.ScrollEvent{
			Event: mouse.Event{
				Where:     where,
				Button:    stb,
				Action:    mouse.Scroll,
				Modifiers: mods,
			},
			Delta: del,
		}
	}
	event.Init()
	lastMouseEvent = event
	w.Send(event)
}

func (w *windowImpl) SetSize(sz image.Point) {
     // todo
}

