// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,!android dragonfly openbsd

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

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/drawer"
	"github.com/goki/gi/oswin/driver/internal/event"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/ki/bitflag"
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

	mu             sync.Mutex
	released       bool
	closeReqFunc   func(win oswin.Window)
	closeCleanFunc func(win oswin.Window)
}

// for sending any kind of event
func sendEvent(w *windowImpl, ev oswin.Event) {
	ev.Init()
	w.Send(ev)
}

// for sending window.Event's
func sendWindowEvent(w *windowImpl, act window.Actions) {
	winEv := window.Event{
		Action: act,
	}
	winEv.Init()
	w.Send(&winEv)
}

func (w *windowImpl) Upload(dp image.Point, src oswin.Image, sr image.Rectangle) {
	src.(*imageImpl).upload(xproto.Drawable(w.xw), w.xg, w.app.xsci.RootDepth, dp, sr)
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
	// todo: support multple screens
	sc := oswin.TheApp.Screen(0)

	dpi := sc.PhysicalDPI
	ldpi := dpi

	sz := image.Point{int(ev.Width), int(ev.Height)}
	ps := image.Point{int(ev.X), int(ev.Y)}

	act := window.ActionsN

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
	w.Scrn = sc
	// }

	sendWindowEvent(w, act)
}

func (w *windowImpl) handleExpose() {
	bitflag.Clear(&w.Flag, int(oswin.Minimized))
	sendWindowEvent(w, window.Paint)
}

func (w *windowImpl) handleKey(detail xproto.Keycode, state uint16, act key.Actions) {
	r, c := w.app.keysyms.Lookup(uint8(detail), state)

	event := &key.Event{
		Rune:      r,
		Code:      c,
		Modifiers: KeyModifiers(state),
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

func (w *windowImpl) handleMouse(x, y int16, button xproto.Button, state uint16, dir mouse.Actions) {
	where := image.Point{int(x), int(y)}
	from := image.ZP
	if lastMouseEvent != nil {
		from = lastMouseEvent.Pos()
	}
	mods := KeyModifiers(state)
	stb := mouse.Buttons(ButtonFromState(state))

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
		act := mouse.Actions(dir)
		if act == mouse.Press && lastMouseClickEvent != nil {
			interval := time.Now().Sub(lastMouseClickEvent.Time())
			// fmt.Printf("interval: %v\n", interval)
			if (interval / time.Millisecond) < time.Duration(mouse.DoubleClickMSec) {
				act = mouse.DoubleClick
			}
		}
		event = &mouse.Event{
			Where:     where,
			Button:    mouse.Buttons(button),
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
	// todo: could used checked

	valmask := uint16(xproto.ConfigWindowWidth + xproto.ConfigWindowHeight)
	vallist := []uint32{uint32(sz.X), uint32(sz.Y)}

	xproto.ConfigureWindow(w.app.xc, w.xw, valmask, vallist)
}

func (w *windowImpl) SetPos(pos image.Point) {
	// todo: could used checked

	valmask := uint16(xproto.ConfigWindowX + xproto.ConfigWindowY)
	vallist := []uint32{uint32(pos.X), uint32(pos.Y)}

	xproto.ConfigureWindow(w.app.xc, w.xw, valmask, vallist)
}

func (w *windowImpl) MainMenu() oswin.MainMenu {
	return nil
}

func (w *windowImpl) Raise() {
	valmask := uint16(xproto.ConfigWindowStackMode)
	vallist := []uint32{uint32(xproto.StackModeAbove)}

	xproto.ConfigureWindow(w.app.xc, w.xw, valmask, vallist)
}

func (w *windowImpl) Minimize() {
	// https://cgit.freedesktop.org/xorg/lib/libX11/tree/src/Iconify.c

	dat := xproto.ClientMessageDataUnionData32New([]uint32{3}) // 3 = IconicState

	minmsg := xproto.ClientMessageEvent{
		Sequence: 1, // no idea what this is..
		Format:   32,
		Window:   w.xw,
		Type:     w.app.atomWMChangeState,
		Data:     dat,
	}

	mask := xproto.EventMaskSubstructureRedirect | xproto.EventMaskSubstructureNotify
	// send to: x.xw
	xproto.SendEvent(w.app.xc, false, w.xw, uint32(mask), string(minmsg.Bytes()))
}

func (w *windowImpl) SetCloseReqFunc(fun func(win oswin.Window)) {
	w.closeReqFunc = fun
}

func (w *windowImpl) SetCloseCleanFunc(fun func(win oswin.Window)) {
	w.closeCleanFunc = fun
}

func (w *windowImpl) CloseReq() {
	if w.closeReqFunc != nil {
		w.closeReqFunc(w)
	} else {
		w.Close()
	}
}

func (w *windowImpl) CloseClean() {
	if w.closeCleanFunc != nil {
		w.closeCleanFunc(w)
	}
}

func (w *windowImpl) Close() {
	xproto.DestroyWindow(w.app.xc, w.xw)
}

func (w *windowImpl) closed() {
	// note: this is the final common path for all window closes
	w.CloseClean()
	sendWindowEvent(w, window.Close)

	w.mu.Lock()
	released := w.released
	w.released = true
	w.mu.Unlock()

	if !released {
		render.FreePicture(w.app.xc, w.xp)
		xproto.FreeGC(w.app.xc, w.xg)
	}

	w.app.DeleteWin(w.xw)
}
