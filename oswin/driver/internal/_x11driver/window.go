// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,!android dragonfly openbsd
// +build !3d

package x11driver

// TODO: implement a back buffer.

import (
	"image"
	"image/color"
	"image/draw"
	"log"
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

	// textures are the textures created for this window -- they are released
	// when the window is closed
	textures map[*textureImpl]struct{}

	// This next group of variables are mutable, but are only modified in the
	// appImpl.run goroutine.

	mu             sync.Mutex
	released       bool
	closeReqFunc   func(win oswin.Window)
	closeCleanFunc func(win oswin.Window)
	// frameSizes are sizes of extra stuff from window manager, for converting positions
	// l,r,t,b
	frameSizes [4]int
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

func (w *windowImpl) getFrameSizes() [4]int {
	if w.frameSizes[2] != 0 {
		return w.frameSizes
	}
	prop, err := xproto.GetProperty(w.app.xc, false, w.xw, w.app.atomNetFrameExtents, xproto.AtomAny, 0, 4).Reply()
	if err != nil {
		log.Printf("X11 _NET_FRAME_EXTENTS Read Property error: %v\n", err)
	}
	if prop.Format == 32 && prop.ValueLen == 4 {
		for i := 0; i < 4; i++ {
			w.frameSizes[i] = int(xgb.Get32(prop.Value[i*4:]))
		}
	} else {
		// log.Printf("X11 _NET_FRAME_EXTENTS Property values not as expected. fmt: %v, len: %v\n", prop.Format, prop.ValueLen)
	}
	return w.frameSizes
}

// note: this does NOT seem result in accurate results compared to event, but
// frame sizes are accurate
func (w *windowImpl) getCurGeom() (pos, size image.Point, borderWidth int, err error) {
	geo, err := xproto.GetGeometry(w.app.xc, xproto.Drawable(w.xw)).Reply()
	if err != nil {
		log.Println(err)
		return
	}
	trpos, err := xproto.TranslateCoordinates(w.app.xc, w.xw, w.app.xsci.Root, geo.X, geo.Y).Reply()
	if err != nil {
		log.Println(err)
		return
	}
	borderWidth = int(geo.BorderWidth)
	frext := w.getFrameSizes() // l,r,t,b
	pos = image.Point{int(trpos.DstX) - frext[0] - 20 + borderWidth, int(trpos.DstY) - frext[2] - 48 + borderWidth}
	size = image.Point{int(geo.Width), int(geo.Height)}
	// fmt.Printf("computed geom, pos: %v size: %v  frext: %v\n", pos, size, frext)
	return
}

func AbsInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func (w *windowImpl) handleConfigureNotify(ev xproto.ConfigureNotifyEvent) {
	// todo: support multiple screens
	sc := oswin.TheApp.Screen(0)
	dpi := sc.PhysicalDPI

	sz := image.Point{int(ev.Width), int(ev.Height)}
	ps := image.Point{int(ev.X), int(ev.Y)}

	w.mu.Lock()
	frext := w.getFrameSizes() // l,r,t,b
	ps.Y -= frext[2]
	ps.X -= frext[0]
	// orgPos := ps

	cpos, _, borderWidth, _ := w.getCurGeom()
	posdif := ps.Sub(cpos)
	// getting erroneous values from event for first event, which is then saved..
	usecp := AbsInt(posdif.X) > 20 || AbsInt(posdif.Y) > 20
	if usecp {
		ps = cpos
	} else {
		ps.X += borderWidth
		ps.Y += borderWidth
	}

	// fmt.Printf("event geom, pos: %v size: %v  cur: %v  posdif: %v  border: %v\n", orgPos, sz, cpos, posdif, borderWidth)
	act := window.Resize

	if w.Sz != sz || w.PhysDPI != dpi {
		act = window.Resize
	} else if w.Pos != ps {
		act = window.Move
		// fmt.Printf("sent mv from: %v\n", w.Pos)
		w.Pos = ps
	}

	w.Sz = sz
	w.PhysDPI = dpi

	// if scrno > 0 && len(theApp.screens) > int(scrno) {
	w.Scrn = sc
	// }
	w.mu.Unlock()

	// fmt.Printf("sending window event: %v: sz: %v pos: %v\n", act, sz, ps)
	sendWindowEvent(w, act)
}

func (w *windowImpl) handleExpose() {
	w.mu.Lock()
	bitflag.Clear(&w.Flag, int(oswin.Minimized))
	w.mu.Unlock()
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

var lastMouseClickTime time.Time
var lastMousePos image.Point

func (w *windowImpl) handleMouse(x, y int16, button xproto.Button, state uint16, dir mouse.Actions) {
	where := image.Point{int(x), int(y)}
	from := lastMousePos
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
		if act == mouse.Press {
			interval := time.Now().Sub(lastMouseClickTime)
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
			lastMouseClickTime = event.Time()
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
	lastMousePos = event.Pos()
	w.Send(event)
}

func (w *windowImpl) Screen() *oswin.Screen {
	w.mu.Lock()
	if w.Scrn == nil { // not sure how that is happening..
		w.Scrn = theApp.Screen(0)
	}
	sc := w.Scrn
	w.mu.Unlock()
	return sc
}

func (w *windowImpl) Size() image.Point {
	w.mu.Lock()
	sz := w.Sz
	w.mu.Unlock()
	return sz
}

func (w *windowImpl) Position() image.Point {
	w.mu.Lock()
	ps := w.Pos
	w.mu.Unlock()
	return ps
}

func (w *windowImpl) PhysicalDPI() float32 {
	w.mu.Lock()
	dpi := w.PhysDPI
	w.mu.Unlock()
	return dpi
}

func (w *windowImpl) LogicalDPI() float32 {
	w.mu.Lock()
	dpi := w.LogDPI
	w.mu.Unlock()
	return dpi
}

func (w *windowImpl) SetLogicalDPI(dpi float32) {
	w.mu.Lock()
	w.LogDPI = dpi
	w.mu.Unlock()
}

func (w *windowImpl) SetTitle(title string) {
	w.Titl = title
	xproto.ChangeProperty(w.app.xc, xproto.PropModeReplace, w.xw, w.app.atomNETWMName, w.app.atomUTF8String, 8, uint32(len(title)), []byte(title))
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

func (w *windowImpl) SetGeom(pos image.Point, sz image.Point) {
	// apparently order is x,y,w,h, then border width -- in numeric order according to xproto.go
	valmask := uint16(xproto.ConfigWindowX + xproto.ConfigWindowY + xproto.ConfigWindowWidth + xproto.ConfigWindowHeight)
	vallist := []uint32{uint32(pos.X), uint32(pos.Y), uint32(sz.X), uint32(sz.Y)}
	xproto.ConfigureWindow(w.app.xc, w.xw, valmask, vallist)
}

func (w *windowImpl) MainMenu() oswin.MainMenu {
	return nil
}

func (w *windowImpl) Raise() {
	// throwing everything at it:
	// https://stackoverflow.com/questions/30192347/how-to-restore-a-window-with-xlib

	xproto.MapWindow(w.app.xc, w.xw)

	vdat := []uint32{1, xproto.TimeCurrentTime, 0, 0, 0} // 1 = make it active somehow
	dat := xproto.ClientMessageDataUnionData32New(vdat)

	minmsg := xproto.ClientMessageEvent{
		Sequence: 0, // no idea what this is..
		Format:   32,
		Window:   w.xw,
		Type:     w.app.atomNetActiveWindow,
		Data:     dat,
	}
	mask := xproto.EventMaskSubstructureRedirect | xproto.EventMaskSubstructureNotify
	// send to: x.xw
	xproto.SendEvent(w.app.xc, true, w.xw, uint32(mask), string(minmsg.Bytes()))

	valmask := uint16(xproto.ConfigWindowStackMode)
	vallist := []uint32{uint32(xproto.StackModeAbove)}

	xproto.ConfigureWindow(w.app.xc, w.xw, valmask, vallist)
}

func (w *windowImpl) Minimize() {
	// https://cgit.freedesktop.org/xorg/lib/libX11/tree/src/Iconify.c

	vdat := []uint32{3, 0, 0, 0, 0} // 3 = IconicState
	dat := xproto.ClientMessageDataUnionData32New(vdat)

	minmsg := xproto.ClientMessageEvent{
		Sequence: 0, // no idea what this is..
		Format:   32,
		Window:   w.xw,
		Type:     w.app.atomWMChangeState,
		Data:     dat,
	}

	mask := xproto.EventMaskSubstructureRedirect | xproto.EventMaskSubstructureNotify
	// send to: x.xw
	xproto.SendEvent(w.app.xc, true, w.xw, uint32(mask), string(minmsg.Bytes()))
}

func (w *windowImpl) AddTexture(t *textureImpl) {
	if w.textures == nil {
		w.textures = make(map[*textureImpl]struct{})
	}
	w.textures[t] = struct{}{}
}

// DeleteTexture just deletes it from our list -- does not Release -- is called during t.Release
func (w *windowImpl) DeleteTexture(t *textureImpl) {
	if w.textures == nil {
		return
	}
	delete(w.textures, t)
}

func (w *windowImpl) SetCloseReqFunc(fun func(win oswin.Window)) {
	w.closeReqFunc = fun
}

func (w *windowImpl) SetCloseCleanFunc(fun func(win oswin.Window)) {
	w.closeCleanFunc = fun
}

func (w *windowImpl) CloseReq() {
	if theApp.quitting {
		w.Close()
		return
	}
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

func (w *windowImpl) closeRelease() {
	w.CloseClean()
	sendWindowEvent(w, window.Close)
	render.FreePicture(w.app.xc, w.xp)
	xproto.FreeGC(w.app.xc, w.xg)
	if w.textures != nil {
		for t := range w.textures {
			t.Release() // deletes from map
		}
	}
}

func (w *windowImpl) Close() {
	w.mu.Lock()
	released := w.released
	w.released = true
	w.mu.Unlock()

	// fmt.Printf("w Close(): %v  released: %v\n", w.Nm, released)

	if !released {
		w.closeRelease()
	}
	xproto.DestroyWindow(w.app.xc, w.xw)
}

func (w *windowImpl) closed() {
	// note: this is the final common path for all window closes
	w.mu.Lock()
	released := w.released
	w.released = true
	w.mu.Unlock()

	// fmt.Printf("w closed(): %v  released: %v\n", w.Nm, released)

	if !released {
		w.closeRelease()
	}

	w.app.DeleteWin(w.xw)

	if theApp.quitting {
		// fmt.Printf("win: %v quit closing\n", w.Nm)
		theApp.quitCloseCnt <- struct{}{}
	}
}
